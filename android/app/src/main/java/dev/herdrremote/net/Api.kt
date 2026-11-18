package dev.herdrremote.net

import java.io.IOException
import java.net.InetSocketAddress
import java.security.KeyStore
import java.security.cert.X509Certificate
import java.util.concurrent.TimeUnit
import javax.net.ssl.SSLContext
import javax.net.ssl.SSLSocket
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.serialization.json.Json
import okhttp3.Call
import okhttp3.ConnectionSpec
import okhttp3.EventListener
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import okhttp3.Response
import okhttp3.TlsVersion

class ApiException(
    val status: Int,
    val error: String,
    val reason: String? = null,
    val idleMs: Long? = null,
) : IOException("$status $error${reason?.let { " ($it)" } ?: ""}")

/** No endpoint answered; the view keeps its last data and shows its age. */
class UnreachableException(message: String) : IOException(message)

/**
 * Tag of an action call. [started] is set once OkHttp begins to write the request: from then on the
 * daemon may have received it, so the app never sends it again. Before that (connect refused,
 * unknown host, a timeout before the TLS handshake ended) the next endpoint may be tried.
 */
class SentMark {
    @Volatile var started = false
}

object SentListener : EventListener() {
    override fun requestHeadersStart(call: Call) {
        call.request().tag(SentMark::class.java)?.started = true
    }
}

val json = Json {
    ignoreUnknownKeys = true
    explicitNulls = false
}
private val JSON = "application/json".toMediaType()

/** Builds OkHttp clients that present the Keystore key and pin the Mac's key. TLS 1.3 only. */
object Clients {
    private val tls13 =
        listOf(
            ConnectionSpec.Builder(ConnectionSpec.RESTRICTED_TLS)
                .tlsVersions(TlsVersion.TLS_1_3)
                .build()
        )

    fun sslContext(pin: String, entry: KeyStore.PrivateKeyEntry): SSLContext {
        val ctx = SSLContext.getInstance("TLSv1.3")
        ctx.init(arrayOf(DeviceKeyManager(DeviceKey.ALIAS, entry)), arrayOf(PinnedTrust(pin)), null)
        return ctx
    }

    fun base(pin: String, entry: KeyStore.PrivateKeyEntry): OkHttpClient.Builder {
        val trust = PinnedTrust(pin)
        return OkHttpClient.Builder()
            .sslSocketFactory(sslContext(pin, entry).socketFactory, trust)
            .hostnameVerifier(trust)
            .connectionSpecs(tls13)
            .connectTimeout(10, TimeUnit.SECONDS)
            .readTimeout(15, TimeUnit.SECONDS)
            .writeTimeout(15, TimeUnit.SECONDS)
    }
}

/** The daemon's API (docs/api.md) with the phone-side timeouts of ADR 0005. */
class HerdrApi(private val store: Store, pin: String, entry: KeyStore.PrivateKeyEntry) {
    private val calls = Clients.base(pin, entry).build()
    /**
     * Actions: 150 s to read, because the daemon waits for the Mac to go quiet and for hc to press
     * the key.
     */
    private val actions =
        calls
            .newBuilder()
            .retryOnConnectionFailure(false)
            .readTimeout(150, TimeUnit.SECONDS)
            .eventListener(SentListener)
            .build()
    private val sse = calls.newBuilder().readTimeout(30, TimeUnit.SECONDS).build()
    private val stream = calls.newBuilder().readTimeout(0, TimeUnit.SECONDS).build()

    private fun base(host: String) = "https://$host:${store.port}"

    /**
     * The last endpoint that worked first (even if /v1/hello no longer lists it), then the others
     * in order.
     */
    private fun candidates(): List<String> {
        val all = store.endpoints
        val last = store.lastGood
        return if (last != null) listOf(last) + all.filter { it != last } else all
    }

    private suspend fun <T> withEndpoint(block: (String) -> T): T =
        withContext(Dispatchers.IO) {
            var lastErr: IOException? = null
            for (host in candidates()) {
                try {
                    val out = block(base(host))
                    store.lastGood = host
                    return@withContext out
                } catch (e: ApiException) {
                    store.lastGood = host
                    throw e
                } catch (e: IOException) {
                    lastErr = e
                }
            }
            throw UnreachableException(lastErr?.message ?: "No endpoint is reachable")
        }

    private fun <T> parse(resp: Response, decode: (String) -> T): T {
        resp.use {
            val body = it.body?.string().orEmpty()
            if (!it.isSuccessful) {
                val e = runCatching {
                    json.decodeFromString(ErrorBody.serializer(), body)
                }
                    .getOrDefault(ErrorBody(error = body.take(200)))
                throw ApiException(
                    it.code,
                    e.error.ifEmpty { "HTTP ${it.code}" },
                    e.reason,
                    e.idleMs,
                )
            }
            return decode(body)
        }
    }

    private inline fun <reified T> get(client: OkHttpClient, url: String): T =
        parse(client.newCall(Request.Builder().url(url).build()).execute()) {
            json.decodeFromString<T>(it)
        }

    suspend fun hello(): Hello = withEndpoint { b ->
        val h: Hello = get(calls, "$b/v1/hello")
        store.clockOffsetMs = h.serverTimeMs - System.currentTimeMillis()
        if (h.endpoints.isNotEmpty()) store.endpoints = h.endpoints
        h
    }

    suspend fun spaces(): List<Space> = withEndpoint { get(calls, "$it/v1/spaces") }

    suspend fun usage(): UsageResponse = withEndpoint { get(calls, "$it/v1/usage") }

    suspend fun agents(): List<Agent> = withEndpoint { get(calls, "$it/v1/agents") }

    suspend fun terminal(t: String): Agent = withEndpoint { get(calls, "$it/v1/terminals/$t") }

    suspend fun transcript(t: String, cursor: String?): TranscriptPage = withEndpoint {
        get(
            calls,
            "$it/v1/terminals/$t/transcript" +
                if (cursor.isNullOrEmpty()) "" else "?cursor=$cursor",
        )
    }

    suspend fun prompt(t: String): PromptView = withEndpoint {
        get(calls, "$it/v1/terminals/$t/prompt")
    }

    suspend fun skills(t: String): Skills = withEndpoint {
        get(calls, "$it/v1/terminals/$t/skills")
    }

    /** The Mac's clock, for UUIDv7 request ids. */
    fun now(): Long = System.currentTimeMillis() + store.clockOffsetMs

    suspend fun decide(t: String, req: DecisionRequest): String =
        action("/v1/terminals/$t/decision", json.encodeToString(DecisionRequest.serializer(), req))

    suspend fun message(t: String, req: MessageRequest): String =
        action("/v1/terminals/$t/message", json.encodeToString(MessageRequest.serializer(), req))

    /**
     * Posts one action. The next endpoint is tried only while the request was never sent (see
     * [SentMark]). After that there is no retry of any kind: the failure is thrown as it is, and
     * the phone shows "Outcome unknown".
     */
    private suspend fun action(path: String, body: String): String =
        withContext(Dispatchers.IO) {
            var lastErr: IOException? = null
            for (host in candidates()) {
                val mark = SentMark()
                try {
                    val req =
                        Request.Builder()
                            .url("${base(host)}$path")
                            .post(body.toRequestBody(JSON))
                            .tag(SentMark::class.java, mark)
                            .build()
                    val out =
                        parse(actions.newCall(req).execute()) {
                            json.decodeFromString(ResultBody.serializer(), it).result
                        }
                    store.lastGood = host
                    return@withContext out
                } catch (e: ApiException) {
                    store.lastGood = host
                    throw e
                } catch (e: IOException) {
                    if (mark.started) throw e
                    lastErr = e
                }
            }
            throw UnreachableException(lastErr?.message ?: "No endpoint is reachable")
        }

    /**
     * The SSE stream, as a call the caller runs with [open] and can cancel to end the blocking
     * read.
     */
    fun events(host: String): Call =
        sse.newCall(Request.Builder().url("${base(host)}/v1/events").build())

    /**
     * The live terminal bytes, as a call the caller runs with [open] and can cancel to end the
     * blocking read. No read timeout: an idle terminal sends nothing.
     */
    fun live(host: String, t: String): Call =
        stream.newCall(Request.Builder().url("${base(host)}/v1/terminals/$t/live").build())

    /** Runs a stream call. A reply that is not 2xx becomes an ApiException. */
    fun open(call: Call): Response {
        val resp = call.execute()
        if (!resp.isSuccessful) parse<Unit>(resp) {}
        return resp
    }

    /** Hosts to try for a stream, best first. */
    fun hosts(): List<String> = candidates()

    fun markGood(host: String) {
        store.lastGood = host
    }
}

/** Enrollment on the enrollment port (docs/architecture.md section 3.1, rule 5). */
object Enroll {
    class Result(val deviceId: String)

    /**
     * The Mac's SPKI DER from one pinned TLS handshake with the enrollment port, so the 6-digit
     * code shows before the long-poll. A wrong pin never connects.
     */
    fun serverKey(link: PairingLink, entry: KeyStore.PrivateKeyEntry): ByteArray {
        val ctx = Clients.sslContext(link.pin, entry)
        var lastErr: IOException? = null
        for (host in link.endpoints) {
            try {
                val socket = ctx.socketFactory.createSocket() as SSLSocket
                socket.use {
                    it.connect(InetSocketAddress(host, link.enrollPort), 10_000)
                    it.soTimeout = 10_000
                    it.enabledProtocols = arrayOf("TLSv1.3")
                    it.startHandshake()
                    val leaf =
                        it.session.peerCertificates.firstOrNull() as? X509Certificate
                            ?: throw IOException("no server certificate")
                    if (!PinnedTrust(link.pin).matches(arrayOf(leaf)))
                        throw IOException("pin mismatch")
                    return leaf.publicKey.encoded
                }
            } catch (e: IOException) {
                lastErr = e
            }
        }
        throw UnreachableException(lastErr?.message ?: "The Mac is not reachable")
    }

    /** POST /v1/enroll: long-polls up to 130 s for Allow, Deny or expiry. */
    fun run(link: PairingLink, name: String, entry: KeyStore.PrivateKeyEntry): Result {
        val client =
            Clients.base(link.pin, entry)
                .readTimeout(130, TimeUnit.SECONDS)
                .retryOnConnectionFailure(false)
                .build()
        var lastErr: IOException? = null
        for (host in link.endpoints) {
            try {
                val body =
                    json.encodeToString(EnrollRequest.serializer(), EnrollRequest(link.token, name))
                val resp =
                    client
                        .newCall(
                            Request.Builder()
                                .url("https://$host:${link.enrollPort}/v1/enroll")
                                .post(body.toRequestBody(JSON))
                                .build()
                        )
                        .execute()
                resp.use {
                    val text = it.body?.string().orEmpty()
                    if (!it.isSuccessful) {
                        val e = runCatching {
                            json.decodeFromString(ErrorBody.serializer(), text)
                        }
                            .getOrDefault(ErrorBody(error = text.take(200)))
                        throw ApiException(it.code, e.error, e.reason)
                    }
                    return Result(json.decodeFromString(EnrollResponse.serializer(), text).deviceId)
                }
            } catch (e: ApiException) {
                throw e
            } catch (e: IOException) {
                lastErr = e
            }
        }
        throw UnreachableException(lastErr?.message ?: "The Mac is not reachable")
    }
}
