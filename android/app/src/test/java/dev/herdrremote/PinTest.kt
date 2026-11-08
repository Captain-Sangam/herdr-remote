package dev.herdrremote

import androidx.biometric.BiometricManager
import dev.herdrremote.net.Pairing
import dev.herdrremote.net.PinnedTrust
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Assert.fail
import org.junit.Test
import java.security.Principal
import java.security.cert.Certificate
import java.security.cert.CertificateException
import java.security.cert.CertificateFactory
import java.security.cert.X509Certificate
import javax.net.ssl.SSLSession
import javax.net.ssl.SSLSessionContext
import javax.security.cert.X509Certificate as LegacyX509

class PinTest {
    private val cert: X509Certificate = javaClass.getResourceAsStream("/server-test.crt")!!.use {
        CertificateFactory.getInstance("X.509").generateCertificate(it) as X509Certificate
    }
    private val pin = Pairing.spkiHashB64(cert.publicKey.encoded)

    private fun session(chain: Array<Certificate>?): SSLSession = object : SSLSession {
        override fun getPeerCertificates(): Array<Certificate> = chain ?: throw javax.net.ssl.SSLPeerUnverifiedException("none")
        override fun getId() = ByteArray(0)
        override fun getSessionContext(): SSLSessionContext? = null
        override fun getCreationTime() = 0L
        override fun getLastAccessedTime() = 0L
        override fun invalidate() {}
        override fun isValid() = true
        override fun putValue(name: String?, value: Any?) {}
        override fun getValue(name: String?): Any? = null
        override fun removeValue(name: String?) {}
        override fun getValueNames(): Array<String> = emptyArray()
        override fun getLocalCertificates(): Array<Certificate>? = null
        @Suppress("DEPRECATION")
        override fun getPeerCertificateChain(): Array<LegacyX509> = emptyArray()
        override fun getPeerPrincipal(): Principal? = null
        override fun getLocalPrincipal(): Principal? = null
        override fun getCipherSuite() = "TLS_AES_128_GCM_SHA256"
        override fun getProtocol() = "TLSv1.3"
        override fun getPeerHost() = "10.0.2.2"
        override fun getPeerPort() = 7766
        override fun getPacketBufferSize() = 0
        override fun getApplicationBufferSize() = 0
    }

    @Test
    fun trustManagerAcceptsOnlyThePinnedKey() {
        PinnedTrust(pin).checkServerTrusted(arrayOf(cert), "ECDHE_ECDSA")
        for (bad in listOf("QRnnziLATVClCScEmJy8DsE-pPDZtEf8qE0DItxf0W4", "", pin.dropLast(1), pin.lowercase())) {
            try {
                PinnedTrust(bad).checkServerTrusted(arrayOf(cert), "ECDHE_ECDSA")
                fail("accepted pin $bad")
            } catch (e: CertificateException) {
            }
        }
        try {
            PinnedTrust(pin).checkServerTrusted(emptyArray(), "ECDHE_ECDSA")
            fail("accepted an empty chain")
        } catch (e: CertificateException) {
        }
    }

    @Test
    fun hostnameVerifierIgnoresNamesAndChecksTheKey() {
        val ok = PinnedTrust(pin)
        assertTrue(ok.verify("10.0.2.2", session(arrayOf(cert))))
        assertTrue(ok.verify("not-the-cn.example", session(arrayOf(cert))))
        assertFalse(ok.verify("10.0.2.2", session(null)))
        assertFalse(PinnedTrust("wrongpin").verify("10.0.2.2", session(arrayOf(cert))))
        assertFalse(PinnedTrust(pin).verify("10.0.2.2", session(emptyArray())))
    }

    @Test
    fun gateLocksAfterFiveMinutes() {
        Gate.reset()
        assertTrue(Gate.locked)
        Gate.unlock()
        assertFalse(Gate.locked)
        Gate.onBackground(1_000_000)
        Gate.onForeground(1_000_000 + Gate.LOCK_AFTER_MS - 1)
        assertFalse("under five minutes stays unlocked", Gate.locked)
        Gate.onBackground(2_000_000)
        Gate.onForeground(2_000_000 + Gate.LOCK_AFTER_MS)
        assertTrue("five minutes locks", Gate.locked)
        assertTrue(Gate.shouldLock(null, 0))
        Gate.reset()
    }

    @Test
    fun onlyAPhoneWithNoScreenLockOpensWithoutAPrompt() {
        assertEquals(Gate.Check.PROMPT, Gate.check(BiometricManager.BIOMETRIC_SUCCESS))
        assertEquals(Gate.Check.OPEN, Gate.check(BiometricManager.BIOMETRIC_ERROR_NONE_ENROLLED))
        for (s in listOf(
            BiometricManager.BIOMETRIC_ERROR_NO_HARDWARE, BiometricManager.BIOMETRIC_ERROR_HW_UNAVAILABLE,
            BiometricManager.BIOMETRIC_ERROR_SECURITY_UPDATE_REQUIRED, BiometricManager.BIOMETRIC_ERROR_UNSUPPORTED,
            BiometricManager.BIOMETRIC_STATUS_UNKNOWN, -42,
        )) {
            assertEquals("status $s", Gate.Check.ERROR, Gate.check(s))
        }
    }
}
