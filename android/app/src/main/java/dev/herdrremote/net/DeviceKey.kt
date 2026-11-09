package dev.herdrremote.net

import android.security.keystore.KeyGenParameterSpec
import android.security.keystore.KeyProperties
import java.security.KeyPairGenerator
import java.security.KeyStore
import java.security.spec.ECGenParameterSpec
import java.util.Date
import javax.security.auth.x500.X500Principal

/**
 * The phone's P-256 key in Android Keystore (TDD section 3.7). Not bound to an unlocked device, so
 * alerts work while locked.
 */
object DeviceKey {
    const val ALIAS = "herdr-remote-device"

    private fun keyStore(): KeyStore = KeyStore.getInstance("AndroidKeyStore").apply { load(null) }

    fun ensure(): KeyStore.PrivateKeyEntry {
        val ks = keyStore()
        if (!ks.containsAlias(ALIAS)) {
            val kpg =
                KeyPairGenerator.getInstance(KeyProperties.KEY_ALGORITHM_EC, "AndroidKeyStore")
            kpg.initialize(
                KeyGenParameterSpec.Builder(ALIAS, KeyProperties.PURPOSE_SIGN)
                    .setAlgorithmParameterSpec(ECGenParameterSpec("secp256r1"))
                    .setDigests(KeyProperties.DIGEST_SHA256, KeyProperties.DIGEST_NONE)
                    .setUserAuthenticationRequired(false)
                    .setCertificateSubject(X500Principal("CN=herdr-remote"))
                    .setCertificateNotAfter(
                        Date(System.currentTimeMillis() + 10L * 365 * 24 * 3600 * 1000)
                    )
                    .build()
            )
            kpg.generateKeyPair()
        }
        return ks.getEntry(ALIAS, null) as KeyStore.PrivateKeyEntry
    }

    fun spkiDer(): ByteArray = ensure().certificate.publicKey.encoded

    fun forget() {
        val ks = keyStore()
        if (ks.containsAlias(ALIAS)) ks.deleteEntry(ALIAS)
    }
}
