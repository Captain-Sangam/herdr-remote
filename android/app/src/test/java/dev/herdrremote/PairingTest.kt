package dev.herdrremote

import dev.herdrremote.net.Pairing
import dev.herdrremote.net.PairingException
import dev.herdrremote.net.Uuid7
import java.io.File
import java.util.Base64
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Assert.fail
import org.junit.Test

class PairingTest {
    private val good =
        "herdr-remote://pair?v=1&pin=QRnnziLATVClCScEmJy8DsE-pPDZtEf8qE0DItxf0W4&ep=192.168.1.3,100.64.0.9,mac.tail.ts.net&p=7766&ep_port=7767&t=abcDEF123_-&exp=2000000000"

    @Test
    fun parsesAllFields() {
        val l = Pairing.parse(good, nowSec = 1_900_000_000)
        assertEquals("QRnnziLATVClCScEmJy8DsE-pPDZtEf8qE0DItxf0W4", l.pin)
        assertEquals(listOf("192.168.1.3", "100.64.0.9", "mac.tail.ts.net"), l.endpoints)
        assertEquals(7766, l.port)
        assertEquals(7767, l.enrollPort)
        assertEquals("abcDEF123_-", l.token)
        assertEquals(2_000_000_000L, l.expiresAt)
    }

    private fun expectFailure(text: String, nowSec: Long = 1_900_000_000, contains: String) {
        try {
            Pairing.parse(text, nowSec)
            fail("parsed $text")
        } catch (e: PairingException) {
            assertTrue("${e.message} should mention $contains", e.message!!.contains(contains))
        }
    }

    @Test
    fun refusesBadVersionExpiredAndMissingFields() {
        expectFailure(good.replace("v=1", "v=2"), contains = "newer app")
        expectFailure(good, nowSec = 2_000_000_001, contains = "expired")
        expectFailure(good.replace("&t=abcDEF123_-", ""), contains = "incomplete")
        expectFailure(good.replace("&ep_port=7767", ""), contains = "incomplete")
        expectFailure(
            good.replace("ep=192.168.1.3,100.64.0.9,mac.tail.ts.net", "ep="),
            contains = "incomplete",
        )
        expectFailure(
            good.replace("pin=QRnnziLATVClCScEmJy8DsE-pPDZtEf8qE0DItxf0W4", "pin="),
            contains = "incomplete",
        )
        expectFailure("https://example.com/pair?v=1", contains = "not a Herdr Remote")
        expectFailure("", contains = "not a Herdr Remote")
    }

    @Test
    fun codeMatchesSharedVectors() {
        val file =
            listOf("../../testdata/pairing-vectors.json", "../testdata/pairing-vectors.json")
                .map(::File)
                .first { it.exists() }
        val vectors = Json.parseToJsonElement(file.readText()).jsonObject["vectors"]!!.jsonArray
        assertTrue(vectors.size >= 5)
        val dec = Base64.getUrlDecoder()
        vectors.forEachIndexed { i, el ->
            val v = el.jsonObject
            val code =
                Pairing.code(
                    dec.decode(v["server_spki_b64url"]!!.jsonPrimitive.content),
                    dec.decode(v["client_spki_b64url"]!!.jsonPrimitive.content),
                )
            assertEquals("vector $i", v["code"]!!.jsonPrimitive.content, code)
        }
    }

    @Test
    fun uuid7HasVersionAndTime() {
        val id = Uuid7.make(1_794_122_774_714L)
        assertEquals(36, id.length)
        assertEquals('7', id[14])
        assertTrue(id[19] in "89ab")
        val ms = (id.substring(0, 8) + id.substring(9, 13)).toLong(16)
        assertEquals(1_794_122_774_714L, ms)
    }
}
