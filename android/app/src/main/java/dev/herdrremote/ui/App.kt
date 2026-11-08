package dev.herdrremote.ui

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.material3.Button
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import dev.herdrremote.HerdrRemoteApp
import dev.herdrremote.net.Space

sealed class Screen {
    data object Home : Screen()
    data class SpaceDetail(val space: Space) : Screen()
    data object AllAgents : Screen()
    data class AgentDetail(val address: String) : Screen()
    data object Settings : Screen()
}

@Composable
fun HerdrRemoteTheme(content: @Composable () -> Unit) {
    MaterialTheme(colorScheme = if (isSystemInDarkTheme()) darkColorScheme() else lightColorScheme(), content = content)
}

@Composable
fun LockedScreen(note: String?, onUnlock: () -> Unit) {
    HerdrRemoteTheme {
        Surface(Modifier.fillMaxSize()) {
            Column(Modifier.fillMaxSize().padding(24.dp), verticalArrangement = Arrangement.Center, horizontalAlignment = Alignment.CenterHorizontally) {
                Text("Herdr Remote", style = MaterialTheme.typography.headlineSmall)
                Text("Locked", Modifier.padding(top = 8.dp), color = MaterialTheme.colorScheme.onSurfaceVariant)
                if (note != null) Text(note, Modifier.padding(top = 8.dp), color = MaterialTheme.colorScheme.error)
                Button(onClick = onUnlock, Modifier.padding(top = 16.dp)) { Text("Unlock") }
            }
        }
    }
}

@Composable
fun HerdrRemoteUi(app: HerdrRemoteApp, initialAddress: String? = null) {
    var paired by remember { mutableStateOf(app.store.paired) }
    var screen by remember { mutableStateOf<Screen>(if (initialAddress != null) Screen.AgentDetail(initialAddress) else Screen.Home) }
    HerdrRemoteTheme {
        Surface(Modifier.fillMaxSize()) {
            Box(Modifier.fillMaxSize().statusBarsPadding()) {
                if (!paired) {
                    PairScreen(app = app, onPaired = { paired = true; screen = Screen.Home; app.repository.start() })
                } else {
                    BackHandler(enabled = screen != Screen.Home) { screen = Screen.Home }
                    val home = { screen = Screen.Home }
                    when (val s = screen) {
                        Screen.Home -> SpacesScreen(app, onSpace = { screen = Screen.SpaceDetail(it) }, onAllAgents = { screen = Screen.AllAgents }, onSettings = { screen = Screen.Settings })
                        is Screen.SpaceDetail -> SpaceScreen(app, s.space, onAgent = { screen = Screen.AgentDetail(it) }, onBack = home)
                        Screen.AllAgents -> AllAgentsScreen(app, onAgent = { screen = Screen.AgentDetail(it) }, onBack = home)
                        is Screen.AgentDetail -> AgentScreen(app, s.address, onBack = home)
                        Screen.Settings -> SettingsScreen(app, onForget = { paired = false; screen = Screen.Home }, onBack = home)
                    }
                }
            }
        }
    }
}

@Composable
fun Header(title: String, onBack: (() -> Unit)? = null, action: (@Composable () -> Unit)? = null) {
    Row(Modifier.fillMaxWidth().padding(horizontal = 8.dp, vertical = 4.dp), verticalAlignment = Alignment.CenterVertically) {
        if (onBack != null) TextButton(onClick = onBack) { Text("Back") }
        Text(title, Modifier.weight(1f).padding(horizontal = 8.dp), style = MaterialTheme.typography.titleLarge, maxLines = 1)
        action?.invoke()
    }
}

@Composable
fun AgentScreen(app: HerdrRemoteApp, address: String, onBack: () -> Unit) {
    Column(Modifier.fillMaxSize()) {
        Header(address, onBack = onBack)
        Text("The agent screen comes in the next step.", Modifier.padding(16.dp))
    }
}
