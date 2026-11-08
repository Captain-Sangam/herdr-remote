package dev.herdrremote.net

import java.security.MessageDigest
import java.util.Base64

/** The content of the pairing QR code (TDD section 3.3, rule 4). */
data class PairingLink(
    val pin: String,
    val endpoints: List<String>,
    val port: Int,
    val enrollPort: Int,
    val token: String,
    val expiresAt: Long,
)

class PairingException(message: String) : Exception(message)

object Pairing {
    private const val PREFIX = "herdr-remote://pair?"

    fun parse(text: String, nowSec: Long = System.currentTimeMillis() / 1000): PairingLink {
        val t = text.trim()
        if (!t.startsWith(PREFIX)) throw PairingException("This is not a Herdr Remote pairing code")
        val q = t.removePrefix(PREFIX).split('&').associate {
            val i = it.indexOf('=')
            if (i < 0) it to "" else it.substring(0, i) to it.substring(i + 1)
        }
        if (q["v"] != "1") throw PairingException("This pairing code needs a newer app")
        val pin = q["pin"].orEmpty()
        val eps = q["ep"].orEmpty().split(',').map { it.trim() }.filter { it.isNotEmpty() }
        val port = q["p"]?.toIntOrNull() ?: 0
        val enrollPort = q["ep_port"]?.toIntOrNull() ?: 0
        val token = q["t"].orEmpty()
        val exp = q["exp"]?.toLongOrNull() ?: 0L
        if (pin.isEmpty() || eps.isEmpty() || port !in 1..65535 || enrollPort !in 1..65535 || token.isEmpty() || exp == 0L) {
            throw PairingException("The pairing code is incomplete")
        }
        if (exp <= nowSec) throw PairingException("This pairing code has expired. Click Pair a phone again on the Mac.")
        return PairingLink(pin, eps, port, enrollPort, token, exp)
    }

    /** The 6-digit code both screens show: SHA-256(server SPKI ‖ client SPKI), first 4 bytes big-endian, mod 1e6. */
    fun code(serverSpki: ByteArray, clientSpki: ByteArray): String {
        val h = MessageDigest.getInstance("SHA-256").digest(serverSpki + clientSpki)
        val n = ((h[0].toLong() and 0xff) shl 24) or ((h[1].toLong() and 0xff) shl 16) or ((h[2].toLong() and 0xff) shl 8) or (h[3].toLong() and 0xff)
        return "%06d".format(n % 1_000_000)
    }

    fun spkiHashB64(spkiDer: ByteArray): String =
        Base64.getUrlEncoder().withoutPadding().encodeToString(MessageDigest.getInstance("SHA-256").digest(spkiDer))
}
