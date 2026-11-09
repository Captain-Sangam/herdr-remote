package dev.herdrremote

import android.os.Bundle
import android.view.WindowManager
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.biometric.BiometricManager
import androidx.biometric.BiometricManager.Authenticators.BIOMETRIC_STRONG
import androidx.biometric.BiometricManager.Authenticators.DEVICE_CREDENTIAL
import androidx.biometric.BiometricPrompt
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.core.content.ContextCompat
import androidx.fragment.app.FragmentActivity
import dev.herdrremote.ui.HerdrRemoteUi
import dev.herdrremote.ui.LockedScreen

class MainActivity : FragmentActivity() {
    companion object { const val EXTRA_ADDRESS = "address" }

    private var locked by mutableStateOf(Gate.locked)
    private var gateNote by mutableStateOf<String?>(null)

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        // Nothing from this app in the recents screen or in screenshots.
        window.setFlags(WindowManager.LayoutParams.FLAG_SECURE, WindowManager.LayoutParams.FLAG_SECURE)
        setContent {
            if (locked) LockedScreen(note = gateNote, onUnlock = { authenticate() })
            else HerdrRemoteUi(app = application as HerdrRemoteApp, initialAddress = intent.getStringExtra(EXTRA_ADDRESS))
        }
    }

    /** The process observer in [HerdrRemoteApp] has run by now: Android sends its ON_START during the activity's start. */
    override fun onResume() {
        super.onResume()
        locked = Gate.locked
        if (locked) authenticate()
        val app = application as HerdrRemoteApp
        if (app.store.paired) app.repository.start()
    }

    private fun authenticate() {
        val auth = BIOMETRIC_STRONG or DEVICE_CREDENTIAL
        val status = BiometricManager.from(this).canAuthenticate(auth)
        when (Gate.check(status)) {
            Gate.Check.PROMPT -> {}
            Gate.Check.OPEN -> {
                // No biometric and no screen lock: there is nothing to ask for.
                gateNote = "This phone has no screen lock, so the app cannot ask for one."
                Gate.unlock()
                locked = false
                return
            }
            Gate.Check.ERROR -> {
                // Stay locked. The Unlock button on the locked screen tries again.
                gateNote = "The screen lock cannot be checked right now (code $status). Tap Unlock to try again."
                return
            }
        }
        val prompt = BiometricPrompt(this, ContextCompat.getMainExecutor(this), object : BiometricPrompt.AuthenticationCallback() {
            override fun onAuthenticationSucceeded(result: BiometricPrompt.AuthenticationResult) {
                Gate.unlock()
                locked = false
            }
            override fun onAuthenticationError(errorCode: Int, errString: CharSequence) {
                gateNote = errString.toString()
            }
        })
        prompt.authenticate(BiometricPrompt.PromptInfo.Builder().setTitle("Herdr Remote").setSubtitle("Unlock to see your agents").setAllowedAuthenticators(auth).build())
    }
}
