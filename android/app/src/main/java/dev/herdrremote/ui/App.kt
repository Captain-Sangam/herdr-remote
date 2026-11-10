package dev.herdrremote.ui

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.requiredSize
import androidx.compose.foundation.layout.safeDrawingPadding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.widthIn
import androidx.compose.material3.Icon
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clipToBounds
import androidx.compose.ui.res.colorResource
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import dev.herdrremote.HerdrRemoteApp
import dev.herdrremote.R
import dev.herdrremote.net.Space
import dev.herdrremote.ui.theme.KeyButton
import dev.herdrremote.ui.theme.KeyKind
import dev.herdrremote.ui.theme.NoteKind
import dev.herdrremote.ui.theme.Pane
import dev.herdrremote.ui.theme.PaneNote
import dev.herdrremote.ui.theme.PaneTheme
import dev.herdrremote.ui.theme.TmuxBar

sealed class Screen {
    data object Home : Screen()

    data class SpaceDetail(val space: Space) : Screen()

    data object AllAgents : Screen()

    data class AgentDetail(val address: String) : Screen()

    data object Settings : Screen()
}

/** The top-level screens, in the order of the tmux bar. */
private val TOP = listOf(Screen.Home, Screen.AllAgents, Screen.Settings)

/** The Pane design system: ui/theme and docs/design.md. */
@Composable fun HerdrRemoteTheme(content: @Composable () -> Unit) = PaneTheme(content = content)

/** No agent data here (SR-11): only the app's name and the Unlock key. */
@Composable
fun LockedScreen(note: String?, onUnlock: () -> Unit) {
    HerdrRemoteTheme {
        val c = Pane.colors
        Surface(Modifier.fillMaxSize()) {
            // Top-aligned: the system fingerprint sheet covers the lower half.
            Column(
                Modifier.fillMaxSize()
                    .safeDrawingPadding()
                    .padding(top = 88.dp, start = Pane.space.xl, end = Pane.space.xl),
                verticalArrangement = Arrangement.spacedBy(10.dp),
                horizontalAlignment = Alignment.CenterHorizontally,
            ) {
                // The launcher icon's foreground, cropped to its 72 dp safe zone.
                Box(
                    Modifier.padding(bottom = 10.dp)
                        .size(84.dp)
                        .clipToBounds()
                        .background(colorResource(R.color.ic_launcher_background)),
                    contentAlignment = Alignment.Center,
                ) {
                    Image(
                        painterResource(R.drawable.ic_launcher_foreground),
                        null,
                        Modifier.requiredSize(126.dp),
                    )
                }
                Text("Herdr Remote", style = Pane.type.head, color = c.text)
                Row(
                    horizontalArrangement = Arrangement.spacedBy(6.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Icon(painterResource(R.drawable.ic_lock), null, Modifier.size(16.dp), c.muted)
                    Text("Locked", style = Pane.type.label, color = c.muted)
                }
                if (note != null) PaneNote(note, NoteKind.Error)
                KeyButton(
                    "Unlock",
                    onUnlock,
                    Modifier.padding(top = 14.dp).widthIn(min = 200.dp),
                    kind = KeyKind.Go,
                )
            }
        }
    }
}

@Composable
fun HerdrRemoteUi(app: HerdrRemoteApp, initialAddress: String? = null) {
    var paired by remember { mutableStateOf(app.store.paired) }
    var screen by remember {
        mutableStateOf<Screen>(
            if (initialAddress != null) Screen.AgentDetail(initialAddress) else Screen.Home
        )
    }
    HerdrRemoteTheme {
        Surface(Modifier.fillMaxSize()) {
            Box(Modifier.fillMaxSize().safeDrawingPadding()) {
                if (!paired) {
                    PairScreen(
                        app = app,
                        onPaired = {
                            paired = true
                            screen = Screen.Home
                            app.repository.start()
                        },
                    )
                } else {
                    BackHandler(enabled = screen != Screen.Home) { screen = Screen.Home }
                    val home = { screen = Screen.Home }
                    val onAgent = { address: String -> screen = Screen.AgentDetail(address) }
                    Column(Modifier.fillMaxSize()) {
                        Box(Modifier.weight(1f)) {
                            when (val s = screen) {
                                Screen.Home ->
                                    SpacesScreen(
                                        app,
                                        onSpace = { screen = Screen.SpaceDetail(it) },
                                        onAgent = onAgent,
                                    )
                                is Screen.SpaceDetail ->
                                    SpaceScreen(app, s.space, onAgent = onAgent, onBack = home)
                                Screen.AllAgents -> AllAgentsScreen(app, onAgent = onAgent)
                                is Screen.AgentDetail -> AgentScreen(app, s.address, onBack = home)
                                Screen.Settings ->
                                    SettingsScreen(
                                        app,
                                        onForget = {
                                            paired = false
                                            screen = Screen.Home
                                        },
                                    )
                            }
                        }
                        val tab = TOP.indexOf(screen)
                        if (tab >= 0) {
                            val state by app.repository.state.collectAsStateWithLifecycle()
                            TmuxBar(
                                listOf("spaces", "agents", "settings"),
                                tab,
                                onSelect = { screen = TOP[it] },
                                alerts = mapOf(1 to state.agents.values.count { it.blocked }),
                            )
                        }
                    }
                }
            }
        }
    }
}
