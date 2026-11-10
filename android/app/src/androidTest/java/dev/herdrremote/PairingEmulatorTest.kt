package dev.herdrremote

import androidx.test.ext.junit.runners.AndroidJUnit4
import androidx.test.platform.app.InstrumentationRegistry
import dev.herdrremote.net.DeviceKey
import dev.herdrremote.net.Enroll
import dev.herdrremote.net.HerdrApi
import dev.herdrremote.net.Pairing
import dev.herdrremote.net.Store
import java.io.IOException
import kotlinx.coroutines.runBlocking
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Assert.fail
import org.junit.Assume.assumeTrue
import org.junit.Test
import org.junit.runner.RunWith

/**
 * Pairs with a dev daemon on the host (10.0.2.2), which auto-allows. Run with ./gradlew
 * connectedDebugAndroidTest -Pandroid.testInstrumentationRunnerArguments.pairText=<link> where the
 * link comes from `herdr-remoted pair --json` of that daemon.
 */
@RunWith(AndroidJUnit4::class)
class PairingEmulatorTest {
    @Test
    fun pairsThenPinDecidesTheHandshake() {
        val raw = InstrumentationRegistry.getArguments().getString("pairText")
        assumeTrue("pairText not given", !raw.isNullOrEmpty())
        val link = Pairing.parse(raw!!.replace(Regex("ep=[^&]*"), "ep=10.0.2.2"))
        val context = InstrumentationRegistry.getInstrumentation().targetContext
        val entry = DeviceKey.ensure()
        val serverSpki = Enroll.serverKey(link, entry)
        val code = Pairing.code(serverSpki, entry.certificate.publicKey.encoded)
        assertEquals("the code is known before Allow", 6, code.length)
        assertEquals(Pairing.spkiHashB64(serverSpki), link.pin)
        val res = Enroll.run(link, "emulator", entry)
        assertTrue(res.deviceId.isNotEmpty())
        // A wrong pin never completes the enrollment handshake either.
        try {
            Enroll.serverKey(link.copy(pin = "A" + link.pin.drop(1).reversed()), entry)
            fail("handshake with a wrong pin")
        } catch (e: IOException) {}

        // The right key with no matching name (CN=herdr-remoted, address 10.0.2.2) connects.
        val store = Store(context)
        store.pin = link.pin
        store.endpoints = link.endpoints
        store.port = link.port
        store.deviceId = res.deviceId
        val hello = runBlocking { HerdrApi(store, link.pin, entry).hello() }
        assertTrue(hello.version.isNotEmpty())
        assertTrue(store.lastGood == "10.0.2.2")

        // A wrong pin never connects.
        val wrong = "A" + link.pin.drop(1).reversed()
        try {
            runBlocking { HerdrApi(store, wrong, entry).hello() }
            fail("connected with a wrong pin")
        } catch (e: IOException) {}
        if (InstrumentationRegistry.getArguments().getString("keep") == "1") {
            // Leave the app paired and watching, for the manual checks on the emulator.
            store.macName = "10.0.2.2"
            store.watch = true
            dev.herdrremote.watch.WatchService.start(context)
        } else {
            store.forget()
        }
    }
}
