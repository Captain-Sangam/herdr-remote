package dev.herdrremote.ui

import android.os.Build
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
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
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

sealed class PairState {
    data object Idle : PairState()
    data object Connecting : PairState()
    data class Code(val code: String) : PairState()
    data class Failed(val message: String) : PairState()
}

/** Pairing (TDD section 3.3): scan or paste the code, show the 6 digits, wait for Allow on the Mac. */
@Composable
fun PairScreen(app: HerdrRemoteApp, onPaired: () -> Unit) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    var state by remember { mutableStateOf<PairState>(PairState.Idle) }
    var pasted by remember { mutableStateOf("") }
    var name by remember { mutableStateOf(Build.MODEL.take(64)) }

    fun start(text: String) {
        val link: PairingLink = try {
            Pairing.parse(text)
        } catch (e: PairingException) {
            state = PairState.Failed(e.message ?: "Bad pairing code"); return
        }
        if (name.isBlank() || name.length > 64) {
            state = PairState.Failed("Give this phone a name of 1 to 64 characters"); return
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
                state = PairState.Failed(
                    when (e.status) {
                        403 -> if (e.error == "denied") "The Mac denied this phone." else "The Mac refused this code: ${e.error}."
                        410 -> "The pairing window closed before Allow. Click Pair a phone again on the Mac."
                        else -> "The Mac answered ${e.status}: ${e.error}"
                    },
                )
            } catch (e: Exception) {
                state = PairState.Failed("Could not reach the Mac: ${e.message ?: e.javaClass.simpleName}")
            }
        }
    }

    fun scan() {
        val options = GmsBarcodeScannerOptions.Builder().setBarcodeFormats(Barcode.FORMAT_QR_CODE).build()
        GmsBarcodeScanning.getClient(context, options).startScan()
            .addOnSuccessListener { barcode -> barcode.rawValue?.let { start(it) } ?: run { state = PairState.Failed("No code was read") } }
            .addOnFailureListener { e -> state = PairState.Failed("Scanner unavailable: ${e.message}") }
    }

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState()).padding(24.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
        Text("Pair with your Mac", style = MaterialTheme.typography.headlineSmall)
        Text("On the Mac, open Herdr Remote and click Pair a phone. Then scan the code it shows.", color = MaterialTheme.colorScheme.onSurfaceVariant)
        when (val s = state) {
            is PairState.Connecting -> Row { CircularProgressIndicator(Modifier.padding(end = 12.dp)); Text("Connecting to the Mac…") }
            is PairState.Code -> {
                Text(s.code, style = MaterialTheme.typography.displayMedium)
                Text("If the Mac shows the same code, click Allow there. Otherwise click Deny.", color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
            is PairState.Failed -> Text(s.message, color = MaterialTheme.colorScheme.error)
            PairState.Idle -> {}
        }
        if (state !is PairState.Connecting && state !is PairState.Code) {
            OutlinedTextField(value = name, onValueChange = { name = it }, label = { Text("This phone's name") }, singleLine = true, modifier = Modifier.fillMaxWidth())
            Button(onClick = { scan() }, Modifier.fillMaxWidth()) { Text("Scan the QR code") }
            Spacer(Modifier.height(8.dp))
            OutlinedTextField(value = pasted, onValueChange = { pasted = it }, label = { Text("Or paste the pairing link") }, modifier = Modifier.fillMaxWidth(), minLines = 2)
            OutlinedButton(onClick = { start(pasted) }, Modifier.fillMaxWidth(), enabled = pasted.isNotBlank()) { Text("Pair with the pasted link") }
        }
    }
}
