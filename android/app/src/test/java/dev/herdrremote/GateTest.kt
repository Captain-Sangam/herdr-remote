package dev.herdrremote

import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleOwner
import androidx.lifecycle.LifecycleRegistry
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test

/** The app gate (decision 14) and its one process-lifecycle observer (review 2, fix 1). */
class GateTest {
    private class Process : LifecycleOwner {
        val registry = LifecycleRegistry.createUnsafe(this)
        override val lifecycle: Lifecycle
            get() = registry
    }

    @Before fun fresh() = Gate.reset()

    @Test
    fun lockedAtStartAndAfterFiveMinutesInTheBackground() {
        assertTrue(Gate.locked)
        Gate.unlock()
        assertFalse(Gate.locked)
        Gate.onBackground(1_000)
        Gate.onForeground(1_000 + Gate.LOCK_AFTER_MS - 1)
        assertFalse("under the limit", Gate.locked)
        Gate.onBackground(5_000)
        Gate.onForeground(5_000 + Gate.LOCK_AFTER_MS)
        assertTrue("at the limit", Gate.locked)
    }

    @Test
    fun theProcessObserverLocksTheGateWithoutAnActivity() {
        var now = 0L
        val process = Process()
        process.registry.addObserver(Gate.Observer { now })
        process.registry.handleLifecycleEvent(Lifecycle.Event.ON_START)
        Gate.unlock()
        // The activity is gone (a swipe from recents); only the process reports the stop, 700 ms
        // later.
        now = 700
        process.registry.handleLifecycleEvent(Lifecycle.Event.ON_STOP)
        now = 700 + Gate.LOCK_AFTER_MS
        process.registry.handleLifecycleEvent(Lifecycle.Event.ON_START)
        assertTrue(Gate.locked)
    }

    @Test
    fun aShortBackgroundKeepsTheGateOpen() {
        var now = 0L
        val process = Process()
        process.registry.addObserver(Gate.Observer { now })
        process.registry.handleLifecycleEvent(Lifecycle.Event.ON_START)
        Gate.unlock()
        now = 1_000
        process.registry.handleLifecycleEvent(Lifecycle.Event.ON_STOP)
        now = 2_000
        process.registry.handleLifecycleEvent(Lifecycle.Event.ON_START)
        assertFalse(Gate.locked)
    }
}
