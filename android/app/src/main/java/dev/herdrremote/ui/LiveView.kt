package dev.herdrremote.ui

import android.annotation.SuppressLint
import android.content.Context
import android.util.Base64
import android.webkit.JavascriptInterface
import android.webkit.WebResourceRequest
import android.webkit.WebResourceResponse
import android.webkit.WebView
import android.webkit.WebViewClient
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.toArgb
import androidx.compose.ui.viewinterop.AndroidView
import androidx.webkit.WebViewAssetLoader
import dev.herdrremote.HerdrRemoteApp
import dev.herdrremote.net.HerdrApi
import dev.herdrremote.ui.theme.Pane
import java.io.ByteArrayInputStream
import java.io.IOException
import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.channels.Channel
import kotlinx.coroutines.delay
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import kotlinx.coroutines.withTimeoutOrNull
import okhttp3.Call

const val LIVE_PAGE = "https://appassets.androidplatform.net/assets/xterm/live.html"
const val CHUNK = 64 * 1024
const val MAX_PENDING = 1024 * 1024

/**
 * Serves only the bundled assets. Every other request gets 403 and no navigation leaves the page.
 */
class AssetOnlyClient(context: Context, private val onReady: () -> Unit = {}) : WebViewClient() {
    private val loader =
        WebViewAssetLoader.Builder()
            .addPathHandler("/assets/", WebViewAssetLoader.AssetsPathHandler(context))
            .build()

    override fun onPageFinished(view: WebView?, url: String?) {
        onReady()
    }

    fun intercept(url: android.net.Uri): WebResourceResponse =
        loader.shouldInterceptRequest(url)
            ?: WebResourceResponse(
                "text/plain",
                "utf-8",
                403,
                "Blocked",
                emptyMap(),
                ByteArrayInputStream(ByteArray(0)),
            )

    override fun shouldInterceptRequest(
        view: WebView?,
        request: WebResourceRequest,
    ): WebResourceResponse = intercept(request.url)

    override fun shouldOverrideUrlLoading(view: WebView?, request: WebResourceRequest?): Boolean =
        true
}

/**
 * The live terminal: xterm.js at the terminal's own size, fed from /v1/terminals/{t}/live with the
 * flow control of ADR 0008. It reopens the stream when it ends (a resize on the Mac), on an SSE
 * reconnect, and when more than 1 MiB is waiting. stop() and reopen() cancel the stream's call, so
 * the blocking read ends, the response closes and the Mac's attach ends.
 */
class LiveFeed(
    private val app: HerdrRemoteApp,
    private val address: String,
    private val web: WebView,
) {
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)
    private val acks = Channel<Int>(Channel.UNLIMITED)
    private var job: Job? = null
    private var call: Call? = null
    @Volatile private var generation = 0
    /** Completed once live.html has loaded; nothing is written before that. */
    val ready = CompletableDeferred<Unit>()

    inner class Bridge {
        /** The only thing the page ever says: how many bytes xterm.js has drawn. */
        @JavascriptInterface
        fun written(n: Int) {
            acks.trySend(n)
        }
    }

    fun start() {
        scope.launch { app.repository.reconnected.collect { reopen() } }
        reopen()
    }

    @Synchronized
    fun stop() {
        scope.cancel()
        call?.cancel()
    }

    @Synchronized
    fun reopen() {
        generation++
        job?.cancel()
        call?.cancel()
        job = scope.launch { run(generation) }
    }

    /** The stream call of this generation, or null when stop() or reopen() came first. */
    @Synchronized
    private fun openCall(api: HerdrApi, host: String, gen: Int): Call? =
        if (gen != generation || !scope.isActive) null
        else api.live(host, address).also { call = it }

    private suspend fun run(gen: Int) {
        val api = app.api() ?: return
        ready.await()
        var backoff = 500L
        while (scope.isActive && gen == generation) {
            try {
                val term = api.terminal(address)
                withContext(Dispatchers.Main) {
                    web.evaluateJavascript(
                        "hrOpen(${term.cols.coerceIn(2, 500)}, ${term.rows.coerceIn(2, 300)})",
                        null,
                    )
                }
                while (acks.tryReceive().isSuccess) {}
                stream(api, gen)
                backoff = 500L
            } catch (e: IOException) {
                // next try
            }
            delay(backoff)
            backoff = minOf(backoff * 2, 10_000L)
        }
    }

    private suspend fun stream(api: HerdrApi, gen: Int) {
        for (host in api.hosts()) {
            val c = openCall(api, host, gen) ?: return
            try {
                api.open(c).use { resp ->
                    api.markGood(host)
                    val src = resp.body?.source() ?: return
                    val buf = ByteArray(CHUNK)
                    var pending = 0L
                    while (gen == generation) {
                        val n = src.read(buf, 0, buf.size)
                        if (n < 0) return
                        pending += n
                        if (pending > MAX_PENDING) {
                            // The phone fell behind: drop the backlog and take a fresh paint.
                            reopen()
                            return
                        }
                        val b64 = Base64.encodeToString(buf, 0, n, Base64.NO_WRAP)
                        withContext(Dispatchers.Main) {
                            web.evaluateJavascript("hrWrite('$b64')", null)
                        }
                        val acked = withTimeoutOrNull(5_000) { acks.receive() }
                        if (acked != null) pending -= acked
                    }
                }
                return
            } catch (e: IOException) {
                if (c.isCanceled())
                    return // the user left, or a reopen took over: not the next host
            }
        }
        throw IOException("no endpoint")
    }
}

@SuppressLint("SetJavaScriptEnabled")
@Composable
fun LiveView(app: HerdrRemoteApp, address: String, modifier: Modifier = Modifier) {
    AndroidView(
        modifier = modifier,
        factory = { context ->
            WebView(context).apply {
                // The terminal colour before live.html paints, not white.
                setBackgroundColor(Pane.terminalBg.toArgb())
                settings.javaScriptEnabled = true
                settings.allowFileAccess = false
                settings.allowContentAccess = false
                settings.domStorageEnabled = false
                settings.setSupportZoom(true)
                settings.builtInZoomControls = true
                settings.displayZoomControls = false
                settings.useWideViewPort = true
                settings.loadWithOverviewMode = true
                val feed = LiveFeed(app, address, this)
                webViewClient = AssetOnlyClient(context) { feed.ready.complete(Unit) }
                addJavascriptInterface(feed.Bridge(), "hr")
                tag = feed
                loadUrl(LIVE_PAGE)
                feed.start()
            }
        },
        onRelease = { web ->
            (web.tag as? LiveFeed)?.stop()
            web.destroy()
        },
    )
}
