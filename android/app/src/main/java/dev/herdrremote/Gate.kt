package dev.herdrremote

import androidx.biometric.BiometricManager

/**
 * The app gate (decision 14): locked at start and after 5 minutes in the
 * background. Nothing is shown and no action runs while locked.
 */
object Gate {
    const val LOCK_AFTER_MS = 5 * 60 * 1000L

    @Volatile var locked: Boolean = true
        private set
    private var backgroundedAt: Long? = null

    fun unlock() { locked = false; backgroundedAt = null }

    fun onBackground(nowMs: Long) { backgroundedAt = nowMs }

    fun onForeground(nowMs: Long) {
        if (shouldLock(backgroundedAt, nowMs)) locked = true
        backgroundedAt = null
    }

    fun shouldLock(backgroundedAt: Long?, nowMs: Long): Boolean =
        locked || (backgroundedAt != null && nowMs - backgroundedAt >= LOCK_AFTER_MS)

    /** What the gate does with a BiometricManager.canAuthenticate() result. */
    enum class Check { PROMPT, OPEN, ERROR }

    /** Only a phone with no screen lock at all opens without a prompt. Any other error keeps the gate locked. */
    fun check(status: Int): Check = when (status) {
        BiometricManager.BIOMETRIC_SUCCESS -> Check.PROMPT
        BiometricManager.BIOMETRIC_ERROR_NONE_ENROLLED -> Check.OPEN
        else -> Check.ERROR
    }

    /** For tests. */
    fun reset() { locked = true; backgroundedAt = null }
}
