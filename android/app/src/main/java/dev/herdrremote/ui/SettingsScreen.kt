package dev.herdrremote.ui

import android.Manifest
import android.content.Intent
import android.content.pm.PackageManager
import android.net.Uri
import android.os.PowerManager
import android.provider.Settings
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.annotation.DrawableRes
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.unit.dp
import androidx.core.content.ContextCompat
import dev.herdrremote.HerdrRemoteApp
import dev.herdrremote.R
import dev.herdrremote.ui.theme.KeyButton
import dev.herdrremote.ui.theme.KeyKind
import dev.herdrremote.ui.theme.NoteKind
import dev.herdrremote.ui.theme.Pane
import dev.herdrremote.ui.theme.PaneHeader
import dev.herdrremote.ui.theme.PaneNote
import dev.herdrremote.ui.theme.PaneSwitch
import dev.herdrremote.watch.WatchService

/**
 * How the phone reaches an endpoint: the daemon lists private addresses, then Tailscale ones (a
 * 100.64.0.0/10 address or the MagicDNS name).
 */
fun endpointKind(endpoint: String): String {
    val o = endpoint.split('.')
    val cgnat = o.size == 4 && o[0] == "100" && (o[1].toIntOrNull() ?: 0) in 64..127
    return if (cgnat || endpoint.endsWith(".ts.net")) "Tailscale" else "Wi-Fi"
}

@Composable
fun SettingsScreen(app: HerdrRemoteApp, onForget: () -> Unit) {
    Column(Modifier.fillMaxSize()) {
        PaneHeader("~/settings")
        Column(
            Modifier.weight(1f)
                .verticalScroll(rememberScrollState())
                .padding(bottom = Pane.space.xl),
            verticalArrangement = Arrangement.spacedBy(Pane.space.s),
        ) {
            ListPane("paired mac") {
                if (app.store.endpoints.isEmpty()) SettingRow(R.drawable.ic_mac, "none")
                app.store.endpoints.forEach {
                    val kind = endpointKind(it)
                    SettingRow(
                        if (kind == "Tailscale") R.drawable.ic_tail else R.drawable.ic_mac,
                        it,
                        kind,
                    )
                }
                SettingRow(R.drawable.ic_key, "Device id", app.store.deviceId ?: "-")
            }
            WatchSettings(app)
            KeyButton(
                "Forget this Mac",
                {
                    app.forgetMac()
                    onForget()
                },
                Modifier.fillMaxWidth()
                    .padding(start = Pane.space.s, end = Pane.space.s, top = Pane.space.s),
                kind = KeyKind.Stop,
            )
            Hint("The Mac keeps this phone listed until you revoke it there.")
        }
    }
}

/** A settings row: an accent icon, the value, a caption under it and an optional control. */
@Composable
fun SettingRow(
    @DrawableRes icon: Int,
    title: String,
    meta: String? = null,
    trailing: @Composable () -> Unit = {},
) {
    Row(
        Modifier.fillMaxWidth()
            .heightIn(min = Pane.space.touch)
            .padding(horizontal = Pane.space.m, vertical = Pane.space.xs),
        horizontalArrangement = Arrangement.spacedBy(Pane.space.m),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(painterResource(icon), null, Modifier.size(18.dp), Pane.colors.accent)
        Column(Modifier.weight(1f)) {
            Text(title, style = Pane.type.title.copy(fontSize = Pane.type.body.fontSize))
            if (meta != null) Text(meta, style = Pane.type.caption, color = Pane.colors.muted)
        }
        trailing()
    }
}

@Composable
fun Hint(text: String) {
    Text(
        text,
        Modifier.padding(horizontal = Pane.space.l),
        style = Pane.type.caption,
        color = Pane.colors.muted,
    )
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
    ListPane("background watching") {
        SettingRow(R.drawable.ic_bell, "Notify me when an agent is blocked") {
            PaneSwitch(
                checked = watch,
                onCheckedChange = { on ->
                    if (!on) {
                        app.store.watch = false
                        watch = false
                        note = null
                        WatchService.stop(context)
                        return@PaneSwitch
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
    }
    note?.let { PaneNote(it, NoteKind.Error, Modifier.padding(horizontal = Pane.space.s)) }
    Hint(
        "Alerts come only while this phone can reach the Mac. The notification never shows the command."
    )
}
