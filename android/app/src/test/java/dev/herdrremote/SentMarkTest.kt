package dev.herdrremote

import dev.herdrremote.net.SentListener
import dev.herdrremote.net.SentMark
import java.io.IOException
import java.net.ServerSocket
import java.util.concurrent.TimeUnit
import kotlin.concurrent.thread
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Assert.fail
import org.junit.Test

/** The mark that decides whether an action may move to the next endpoint (fix 1). */
class SentMarkTest {
    private fun post(port: Int, readMs: Long): SentMark {
        val client =
            OkHttpClient.Builder()
                .eventListener(SentListener)
                .retryOnConnectionFailure(false)
                .connectTimeout(2, TimeUnit.SECONDS)
                .readTimeout(readMs, TimeUnit.MILLISECONDS)
                .build()
        val mark = SentMark()
        val req =
            Request.Builder()
                .url("http://127.0.0.1:$port/v1/x")
                .post("{}".toRequestBody("application/json".toMediaType()))
                .tag(SentMark::class.java, mark)
                .build()
        try {
            client.newCall(req).execute().close()
            fail("the call succeeded")
        } catch (e: IOException) {}
        return mark
    }

    @Test
    fun aRefusedConnectionWasNeverSent() {
        val port = ServerSocket(0).use { it.localPort } // closed again: nothing listens there
        assertFalse(post(port, 1000).started)
    }

    @Test
    fun aRequestWithNoReplyMayHaveArrived() {
        ServerSocket(0).use { server ->
            val t = thread {
                runCatching {
                    server.accept().use {
                        it.getInputStream().read(ByteArray(4096))
                        Thread.sleep(2000)
                    }
                }
            }
            assertTrue(post(server.localPort, 300).started)
            t.interrupt()
        }
    }
}
