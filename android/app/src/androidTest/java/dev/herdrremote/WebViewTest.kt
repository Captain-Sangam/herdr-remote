package dev.herdrremote

import android.net.Uri
import androidx.test.ext.junit.runners.AndroidJUnit4
import androidx.test.platform.app.InstrumentationRegistry
import dev.herdrremote.ui.AssetOnlyClient
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test
import org.junit.runner.RunWith

@RunWith(AndroidJUnit4::class)
class WebViewTest {
    @Test
    fun liveWebViewRefusesNetworkUrls() {
        val context = InstrumentationRegistry.getInstrumentation().targetContext
        val client = AssetOnlyClient(context)
        for (url in
            listOf(
                "https://example.com/",
                "http://10.0.2.2:7766/v1/agents",
                "file:///etc/hosts",
                "content://media/x",
                "https://appassets.androidplatform.net/res/x",
            )) {
            assertEquals(url, 403, client.intercept(Uri.parse(url)).statusCode)
        }
        // Bundled assets are served (the loader leaves the status at 0; the WebView fills in 200).
        for (asset in listOf("live.html", "xterm.js", "xterm.css")) {
            val r =
                client.intercept(
                    Uri.parse("https://appassets.androidplatform.net/assets/xterm/$asset")
                )
            assertTrue(asset, r.statusCode != 403 && r.data != null)
        }
        assertEquals(
            true,
            client.shouldOverrideUrlLoading(null, null as android.webkit.WebResourceRequest?),
        )
    }
}
