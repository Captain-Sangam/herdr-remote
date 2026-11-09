package dev.herdrremote.ui.theme

import android.app.Activity
import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Shapes
import androidx.compose.material3.Typography
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.Immutable
import androidx.compose.runtime.SideEffect
import androidx.compose.runtime.remember
import androidx.compose.runtime.staticCompositionLocalOf
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalView
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.Font
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.core.view.WindowCompat
import dev.herdrremote.R

/*
 * Pane, the herdr-remote design system (DESIGN.md). The app looks like the terminal it controls:
 * monospace text, square panes with 1 dp lines and ANSI status colours. Every screen reads its
 * colours, type and spacing from here (Pane.colors, Pane.type, Pane.space) and builds on the
 * components in PaneComponents.kt. Do not hard-code a colour or a text size in a screen.
 */

/** The colour tokens. The names match DESIGN.md and design/mockups.js. */
@Immutable
data class PaneColors(
    val bg: Color,
    val surface: Color,
    val surface2: Color,
    val line: Color,
    val text: Color,
    val muted: Color,
    val accent: Color,
    val onAccent: Color,
    val idle: Color,
    val working: Color,
    val done: Color,
    val blocked: Color,
    val error: Color,
    val isDark: Boolean,
) {
    /** The colour of an hc agent state. An unknown state is idle. */
    fun status(state: String): Color =
        when (state) {
            "blocked" -> blocked
            "working" -> working
            "done" -> done
            "error" -> error
            else -> idle
        }
}

val PaneLight =
    PaneColors(
        bg = Color(0xFFEEF1F6),
        surface = Color(0xFFFFFFFF),
        surface2 = Color(0xFFE2E7EF),
        line = Color(0xFFB7C0D1),
        text = Color(0xFF222836),
        muted = Color(0xFF5B667D),
        accent = Color(0xFF2E7D8C),
        onAccent = Color(0xFFFFFFFF),
        idle = Color(0xFF7A8499),
        working = Color(0xFF3A6EA5),
        done = Color(0xFF4F7F2E),
        blocked = Color(0xFFB23A48),
        error = Color(0xFFB4561E),
        isDark = false,
    )

/** The main mode. */
val PaneDark =
    PaneColors(
        bg = Color(0xFF191C26),
        surface = Color(0xFF1F2330),
        surface2 = Color(0xFF272C3B),
        line = Color(0xFF3B4258),
        text = Color(0xFFD8DEE9),
        muted = Color(0xFF7D879E),
        accent = Color(0xFF88C0D0),
        onAccent = Color(0xFF191C26),
        idle = Color(0xFF6C7891),
        working = Color(0xFF81A1C1),
        done = Color(0xFFA3BE8C),
        blocked = Color(0xFFBF616A),
        error = Color(0xFFD08770),
        isDark = true,
    )

/** JetBrains Mono (SIL OFL 1.1, assets/licenses), the only family. Two weights. */
val JetBrainsMono =
    FontFamily(
        Font(R.font.jetbrains_mono_regular, FontWeight.Normal),
        Font(R.font.jetbrains_mono_bold, FontWeight.Bold),
    )

/** The type scale. One family, two weights, sizes and line heights from the mockups. */
@Immutable
data class PaneType(
    /** 44/52 bold: the pairing code. */
    val code: TextStyle,
    /** 18/26 bold: the title of a full-screen flow ("Pair with your Mac"). */
    val display: TextStyle,
    /** 16/24 bold: an empty-state title. */
    val head: TextStyle,
    /** 14/20 bold: the header path and row names. */
    val title: TextStyle,
    /** 13/20: body text, field input. */
    val body: TextStyle,
    /** 12/18: buttons, tabs, pane titles, transcript and code. */
    val label: TextStyle,
    /** 11/16: meta lines, status tags, hints. */
    val caption: TextStyle,
)

// Ligatures off: a command must show exactly as typed ("->" stays two characters).
private fun mono(size: Int, line: Int, bold: Boolean = false) =
    TextStyle(
        fontFamily = JetBrainsMono,
        fontWeight = if (bold) FontWeight.Bold else FontWeight.Normal,
        fontSize = size.sp,
        lineHeight = line.sp,
        fontFeatureSettings = "calt 0, liga 0",
    )

val PaneTypeScale =
    PaneType(
        code = mono(44, 52, bold = true),
        display = mono(18, 26, bold = true),
        head = mono(16, 24, bold = true),
        title = mono(14, 20, bold = true),
        body = mono(13, 20),
        label = mono(12, 18),
        caption = mono(11, 16),
    )

/** Spacing in dp. One character cell is about 8 dp, one text row 20 dp. */
object PaneSpace {
    val xs = 4.dp
    val s = 8.dp
    val m = 10.dp
    val l = 12.dp
    val xl = 16.dp
    /** The minimum size of anything the user taps. */
    val touch = 48.dp
    /** Every border and rule. */
    val line = 1.dp
}

/** No rounded corners anywhere. */
private val Square = RoundedCornerShape(0.dp)
val PaneShapes =
    Shapes(
        extraSmall = Square,
        small = Square,
        medium = Square,
        large = Square,
        extraLarge = Square,
    )

private val LocalPaneColors = staticCompositionLocalOf { PaneDark }

/** Read the tokens in a screen: Pane.colors.blocked, Pane.type.body, Pane.space.l. */
object Pane {
    val colors: PaneColors
        @Composable get() = LocalPaneColors.current

    val type = PaneTypeScale
    val space = PaneSpace
}

/**
 * Material 3 roles for the Pane tokens, so that stock components (Button, TextField, Switch, Card)
 * also look like Pane. Prefer the Pane components for new screens.
 */
private fun PaneColors.toMaterial() =
    (if (isDark) darkColorScheme() else lightColorScheme()).copy(
        primary = accent,
        onPrimary = onAccent,
        primaryContainer = surface2,
        onPrimaryContainer = text,
        inversePrimary = accent,
        secondary = done,
        onSecondary = bg,
        secondaryContainer = surface2,
        onSecondaryContainer = text,
        tertiary = working,
        onTertiary = bg,
        tertiaryContainer = surface2,
        onTertiaryContainer = text,
        background = bg,
        onBackground = text,
        surface = bg,
        onSurface = text,
        surfaceVariant = surface2,
        onSurfaceVariant = muted,
        surfaceTint = bg,
        inverseSurface = text,
        inverseOnSurface = bg,
        error = blocked,
        onError = bg,
        errorContainer = error,
        onErrorContainer = bg,
        outline = line,
        outlineVariant = line,
        surfaceBright = surface2,
        surfaceDim = bg,
        surfaceContainerLowest = bg,
        surfaceContainerLow = surface,
        surfaceContainer = surface,
        surfaceContainerHigh = surface2,
        surfaceContainerHighest = surface2,
    )

private fun PaneType.toMaterial(): Typography {
    val button = label.copy(fontWeight = FontWeight.Bold)
    return Typography(
        displayLarge = code,
        displayMedium = code,
        displaySmall = code,
        headlineLarge = display,
        headlineMedium = display,
        headlineSmall = display,
        titleLarge = head,
        titleMedium = title,
        titleSmall = title,
        bodyLarge = body,
        bodyMedium = body,
        bodySmall = caption,
        labelLarge = button,
        labelMedium = label,
        labelSmall = caption,
    )
}

/** The app theme. Dark is the main mode; light follows the system setting. */
@Composable
fun PaneTheme(dark: Boolean = isSystemInDarkTheme(), content: @Composable () -> Unit) {
    val colors = if (dark) PaneDark else PaneLight
    val view = LocalView.current
    if (!view.isInEditMode) {
        SideEffect {
            val window = (view.context as? Activity)?.window ?: return@SideEffect
            // Light mode needs dark system bar icons, dark mode light ones.
            WindowCompat.getInsetsController(window, view).run {
                isAppearanceLightStatusBars = !dark
                isAppearanceLightNavigationBars = !dark
            }
        }
    }
    val scheme = remember(colors) { colors.toMaterial() }
    val typography = remember { PaneTypeScale.toMaterial() }
    CompositionLocalProvider(LocalPaneColors provides colors) {
        MaterialTheme(
            colorScheme = scheme,
            typography = typography,
            shapes = PaneShapes,
            content = content,
        )
    }
}
