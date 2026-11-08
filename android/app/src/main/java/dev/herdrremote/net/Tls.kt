package dev.herdrremote.net

import java.net.Socket
import java.security.KeyStore
import java.security.MessageDigest
import java.security.Principal
import java.security.PrivateKey
import java.security.cert.Certificate
import java.security.cert.CertificateException
import java.security.cert.X509Certificate
import javax.net.ssl.HostnameVerifier
import javax.net.ssl.SSLSession
import javax.net.ssl.X509ExtendedKeyManager
import javax.net.ssl.X509TrustManager

/**
 * Accepts only a leaf certificate whose SPKI SHA-256 equals the pin. Names and
 * addresses play no part, so LAN and Tailscale endpoints work the same way.
 * It is both the trust manager and the hostname verifier (TDD section 3.7).
 */
class PinnedTrust(private val pin: String) : X509TrustManager, HostnameVerifier {
    fun matches(chain: Array<out Certificate>?): Boolean {
        val leaf = chain?.firstOrNull() as? X509Certificate ?: return false
        val got = Pairing.spkiHashB64(leaf.publicKey.encoded).toByteArray()
        return MessageDigest.isEqual(got, pin.toByteArray())
    }

    override fun checkServerTrusted(chain: Array<X509Certificate>?, authType: String?) {
        if (!matches(chain)) throw CertificateException("The Mac's key does not match the pairing")
    }

    override fun checkClientTrusted(chain: Array<X509Certificate>?, authType: String?) {
        throw CertificateException("not a server")
    }

    override fun getAcceptedIssuers(): Array<X509Certificate> = emptyArray()

    override fun verify(hostname: String?, session: SSLSession?): Boolean =
        try { matches(session?.peerCertificates) } catch (e: Exception) { false }
}

/** Hands the TLS stack the Keystore key: the private key never leaves the Keystore. */
class DeviceKeyManager(private val alias: String, private val entry: KeyStore.PrivateKeyEntry) : X509ExtendedKeyManager() {
    override fun chooseClientAlias(keyType: Array<out String>?, issuers: Array<out Principal>?, socket: Socket?): String = alias
    override fun chooseEngineClientAlias(keyType: Array<out String>?, issuers: Array<out Principal>?, engine: javax.net.ssl.SSLEngine?): String = alias
    override fun getClientAliases(keyType: String?, issuers: Array<out Principal>?): Array<String> = arrayOf(alias)
    override fun getCertificateChain(alias: String?): Array<X509Certificate>? =
        if (alias == this.alias) entry.certificateChain.map { it as X509Certificate }.toTypedArray() else null
    override fun getPrivateKey(alias: String?): PrivateKey? = if (alias == this.alias) entry.privateKey else null
    override fun chooseServerAlias(keyType: String?, issuers: Array<out Principal>?, socket: Socket?): String? = null
    override fun getServerAliases(keyType: String?, issuers: Array<out Principal>?): Array<String>? = null
}
