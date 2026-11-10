package dev.herdrremote.net

import java.security.SecureRandom

/**
 * A UUIDv7 for a request id; the time comes from the caller so clock skew to the Mac can be
 * corrected.
 */
object Uuid7 {
    private val rnd = SecureRandom()

    fun make(nowMs: Long): String {
        val r = ByteArray(10).also { rnd.nextBytes(it) }
        val ms = nowMs and 0xFFFFFFFFFFFFL
        val a = (ms shr 16)
        val b = ms and 0xffff
        val c =
            0x7000L or
                (((r[0].toLong() and 0xff) shl 4) or ((r[1].toLong() and 0xff) shr 4)) and
                0x7fff
        val d = 0x8000L or (((r[2].toLong() and 0x3f) shl 8) or (r[3].toLong() and 0xff))
        var e = 0L
        for (i in 4 until 10) e = (e shl 8) or (r[i].toLong() and 0xff)
        return "%08x-%04x-%04x-%04x-%012x".format(a, b, c, d, e)
    }
}
