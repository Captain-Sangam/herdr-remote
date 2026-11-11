package dev.herdrremote.ui

import android.os.Build
import androidx.compose.foundation.Canvas
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.PathEffect
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.SpanStyle
import androidx.compose.ui.text.buildAnnotatedString
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.withStyle
import androidx.compose.ui.unit.dp
import com.google.mlkit.vision.barcode.common.Barcode
import com.google.mlkit.vision.codescanner.GmsBarcodeScannerOptions
import com.google.mlkit.vision.codescanner.GmsBarcodeScanning
import dev.herdrremote.HerdrRemoteApp
import dev.herdrremote.net.ApiException
import dev.herdrremote.net.DeviceKey
import dev.herdrremote.net.Enroll
import dev.herdrremote.net.Pairing
import dev.herdrremote.net.PairingException
import dev.herdrremote.net.PairingLink
import dev.herdrremote.ui.theme.KeyButton
import dev.herdrremote.ui.theme.KeyKind
import dev.herdrremote.ui.theme.NoteKind
import dev.herdrremote.ui.theme.Pane
import dev.herdrremote.ui.theme.PaneField
import dev.herdrremote.ui.theme.PaneNote
import dev.herdrremote.ui.theme.bottomRule
import dev.herdrremote.ui.theme.spinnerFrame
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

sealed class PairState {
    data object Idle : PairState()

    data object Connecting : PairState()

    data class Code(val code: String) : PairState()

    data class Failed(val message: String) : PairState()
}

/**
 * Pairing (docs/architecture.md section 3.1): scan or paste the code, show the 6 digits, wait for
 * Allow on the Mac.
 */
@Composable
fun PairScreen(app: HerdrRemoteApp, onPaired: () -> Unit) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    var state by remember { mutableStateOf<PairState>(PairState.Idle) }
    var pasted by remember { mutableStateOf("") }
    var name by remember { mutableStateOf(Build.MODEL.take(64)) }

    fun start(text: String) {
        val link: PairingLink =
            try {
                Pairing.parse(text)
            } catch (e: PairingException) {
                state = PairState.Failed(e.message ?: "Bad pairing code")
                return
            }
        if (name.isBlank() || name.length > 64) {
            state = PairState.Failed("Give this phone a name of 1 to 64 characters")
            return
        }
        state = PairState.Connecting
        scope.launch {
            try {
                val entry = withContext(Dispatchers.IO) { DeviceKey.ensure() }
                val clientSpki = entry.certificate.publicKey.encoded
                val serverSpki = withContext(Dispatchers.IO) { Enroll.serverKey(link, entry) }
                state = PairState.Code(Pairing.code(serverSpki, clientSpki))
                val result = withContext(Dispatchers.IO) { Enroll.run(link, name.trim(), entry) }
                with(app.store) {
                    pin = link.pin
                    endpoints = link.endpoints
                    port = link.port
                    deviceId = result.deviceId
                    macName = link.endpoints.firstOrNull().orEmpty()
                }
                runCatching { app.api()?.hello() }
                onPaired()
            } catch (e: ApiException) {
                state =
                    PairState.Failed(
                        when (e.status) {
                            403 ->
                                if (e.error == "denied") "The Mac denied this phone."
                                else "The Mac refused this code: ${e.error}."
                            410 ->
                                "The pairing window closed before Allow. Click Pair a phone again on the Mac."
                            else -> "The Mac answered ${e.status}: ${e.error}"
                        }
                    )
            } catch (e: Exception) {
                state =
                    PairState.Failed(
                        "Could not reach the Mac: ${e.message ?: e.javaClass.simpleName}"
                    )
            }
        }
    }

    fun scan() {
        val options =
            GmsBarcodeScannerOptions.Builder().setBarcodeFormats(Barcode.FORMAT_QR_CODE).build()
        GmsBarcodeScanning.getClient(context, options)
            .startScan()
            .addOnSuccessListener { barcode ->
                barcode.rawValue?.let { start(it) }
                    ?: run { state = PairState.Failed("No code was read") }
            }
            .addOnFailureListener { e ->
                state = PairState.Failed("Scanner unavailable: ${e.message}")
            }
    }

    val c = Pane.colors
    Column(
        Modifier.fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(horizontal = Pane.space.xl, vertical = 18.dp),
        verticalArrangement = Arrangement.spacedBy(14.dp),
    ) {
        PairArt()
        when (val s = state) {
            is PairState.Code -> {
                Title("Check the code")
                Text("The Mac shows a 6-digit code too.", style = Pane.type.body, color = c.muted)
                Row(horizontalArrangement = Arrangement.spacedBy(14.dp)) {
                    s.code.chunked(3).forEach {
                        Text(
                            it,
                            Modifier.border(Pane.space.line, c.line)
                                .padding(horizontal = 10.dp, vertical = 4.dp),
                            style = Pane.type.code,
                            color = c.done,
                        )
                    }
                }
                Text(
                    marked(
                        "If the Mac shows the same code, click **Allow** there. Otherwise click **Deny**."
                    ),
                    style = Pane.type.body,
                    color = c.muted,
                )
                Waiting("Waiting for Allow on the Mac")
                Text("This phone: ${name.trim()}", style = Pane.type.caption, color = c.muted)
            }
            PairState.Connecting -> {
                Title("Pair with your Mac")
                Waiting("Connecting to the Mac…")
            }
            else -> {
                Title("Pair with your Mac")
                if (s is PairState.Failed) PaneNote(s.message, NoteKind.Error)
                else
                    Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                        listOf(
                                "On the Mac, open Herdr Remote and click **Pair a phone**.",
                                "Scan the QR code that the Mac shows.",
                                "If both codes match, click **Allow** on the Mac.",
                            )
                            .forEachIndexed { i, step ->
                                Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                                    Text("${i + 1}.", style = Pane.type.body, color = c.accent)
                                    Text(marked(step), style = Pane.type.body, color = c.text)
                                }
                            }
                    }
                PaneField("This phone's name", name, { name = it })
                KeyButton(
                    if (s is PairState.Failed) "Scan the QR code again" else "Scan the QR code",
                    { scan() },
                    Modifier.fillMaxWidth(),
                    kind = KeyKind.Go,
                )
                Row(
                    horizontalArrangement = Arrangement.spacedBy(10.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Box(
                        Modifier.weight(1f)
                            .height(Pane.space.line)
                            .bottomRule(c.line, dashed = true)
                    )
                    Text("or", style = Pane.type.caption, color = c.muted)
                    Box(
                        Modifier.weight(1f)
                            .height(Pane.space.line)
                            .bottomRule(c.line, dashed = true)
                    )
                }
                PaneField(
                    "Or paste the pairing link",
                    pasted,
                    { pasted = it },
                    placeholder = "herdr-remote://pair?t=…",
                    singleLine = false,
                )
                KeyButton(
                    "Pair with the pasted link",
                    { start(pasted) },
                    Modifier.fillMaxWidth(),
                    enabled = pasted.isNotBlank(),
                )
            }
        }
    }
}

/** "# Pair with your Mac": a display title with an accent "# ". */
@Composable
private fun Title(text: String) {
    Text(
        buildAnnotatedString {
            withStyle(SpanStyle(color = Pane.colors.accent)) { append("# ") }
            append(text)
        },
        style = Pane.type.display,
        color = Pane.colors.text,
    )
}

/** "**Allow**" in bold text colour, the rest as it is. */
@Composable
private fun marked(text: String) = buildAnnotatedString {
    text.split("**").forEachIndexed { i, part ->
        if (i % 2 == 0) append(part)
        else
            withStyle(SpanStyle(fontWeight = FontWeight.Bold, color = Pane.colors.text)) {
                append(part)
            }
    }
}

/** A braille spinner and what the phone waits for. */
@Composable
private fun Waiting(text: String) {
    Row(
        horizontalArrangement = Arrangement.spacedBy(10.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(
            spinnerFrame().toString(),
            style = Pane.type.head,
            color = Pane.colors.accent,
        )
        Text(text, style = Pane.type.label, color = Pane.colors.muted)
    }
}

/** The Mac shows a QR code and the phone scans it (PAIR_ART in design/mockups.js). */
@Composable
private fun PairArt() {
    val c = Pane.colors
    Canvas(Modifier.size(220.dp, 96.dp)) {
        val u = 1.dp.toPx()
        fun at(x: Number, y: Number) = Offset(x.toFloat() * u, y.toFloat() * u)
        fun box(w: Number, h: Number) = Size(w.toFloat() * u, h.toFloat() * u)
        val thin = Stroke(1.6f * u)
        val bold = Stroke(2.5f * u)
        // The Mac and its QR code.
        drawRect(c.text, at(20, 12), box(96, 62), style = thin)
        drawLine(c.text, at(8, 82), at(128, 82), thin.width)
        listOf(50 to 25, 76 to 25, 50 to 51, 76 to 51).forEach { (x, y) ->
            drawRect(c.text, at(x, y), box(10, 10))
        }
        drawRect(c.text, at(65, 40), box(6, 6))
        // The scan, and the phone with its viewfinder.
        drawLine(
            c.accent,
            at(128, 44),
            at(160, 44),
            bold.width,
            pathEffect = PathEffect.dashPathEffect(floatArrayOf(4 * u, 5 * u)),
        )
        drawRect(c.text, at(166, 8), box(44, 80), style = thin)
        drawLine(c.text, at(181, 16), at(195, 16), thin.width)
        drawRect(c.accent, at(176, 36), box(24, 24), style = bold)
    }
}
