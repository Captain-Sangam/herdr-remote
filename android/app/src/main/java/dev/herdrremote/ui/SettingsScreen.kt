package dev.herdrremote.ui

import android.Manifest
import android.content.Intent
import android.content.pm.PackageManager
import android.net.Uri
import android.os.PowerManager
import android.provider.Settings
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import androidx.core.content.ContextCompat
import dev.herdrremote.HerdrRemoteApp
import dev.herdrremote.watch.WatchService

@Composable
fun SettingsScreen(app: HerdrRemoteApp, onForget: () -> Unit, onBack: () -> Unit) {
    Column(Modifier.fillMaxSize()) {
        Header("Settings", onBack = onBack)
        Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
            Text("Paired Mac", style = MaterialTheme.typography.titleMedium)
            Text(
                app.store.endpoints.joinToString(", ").ifEmpty { "none" },
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            Text(
                "Device id ${app.store.deviceId ?: "-"}",
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            WatchSettings(app)
            Button(
                onClick = {
                    app.forgetMac()
                    onForget()
                },
                Modifier.fillMaxWidth(),
                colors =
                    ButtonDefaults.buttonColors(containerColor = MaterialTheme.colorScheme.error),
            ) {
                Text("Forget this Mac")
            }
            Text(
                "The Mac keeps this phone listed until you revoke it there.",
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                style = MaterialTheme.typography.bodySmall,
            )
        }
    }
}

@Composable
fun WatchSettings(app: HerdrRemoteApp) {
    val context = LocalContext.current
    var watch by remember { mutableStateOf(app.store.watch) }
    var note by remember { mutableStateOf<String?>(null) }
    fun turnOn() {
        app.store.watch = true
        watch = true
        note = null
        WatchService.start(context)
        val pm = context.getSystemService(PowerManager::class.java)
        if (!pm.isIgnoringBatteryOptimizations(context.packageName)) {
            runCatching {
                context.startActivity(
                    Intent(
                        Settings.ACTION_REQUEST_IGNORE_BATTERY_OPTIMIZATIONS,
                        Uri.parse("package:${context.packageName}"),
                    )
                )
            }
        }
    }
    val ask =
        rememberLauncherForActivityResult(ActivityResultContracts.RequestPermission()) { granted ->
            if (granted) turnOn()
            else {
                watch = false
                app.store.watch = false
                note =
                    "Notifications are off for this app, so background watching cannot alert you. Allow them in the system settings to turn this on."
            }
        }
    Text("Background watching", style = MaterialTheme.typography.titleMedium)
    Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
        Text("Notify me when an agent is blocked", Modifier.weight(1f))
        Switch(
            checked = watch,
            onCheckedChange = { on ->
                if (!on) {
                    app.store.watch = false
                    watch = false
                    note = null
                    WatchService.stop(context)
                    return@Switch
                }
                if (
                    ContextCompat.checkSelfPermission(
                        context,
                        Manifest.permission.POST_NOTIFICATIONS,
                    ) == PackageManager.PERMISSION_GRANTED
                )
                    turnOn()
                else ask.launch(Manifest.permission.POST_NOTIFICATIONS)
            },
        )
    }
    if (note != null)
        Text(
            note!!,
            color = MaterialTheme.colorScheme.error,
            style = MaterialTheme.typography.bodySmall,
        )
    Text(
        "Alerts come only while this phone can reach the Mac. The notification never shows the command.",
        color = MaterialTheme.colorScheme.onSurfaceVariant,
        style = MaterialTheme.typography.bodySmall,
    )
}
