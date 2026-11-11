package dev.herdrremote.net

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class Agent(
    val address: String,
    val space: String = "",
    val tab: String = "",
    val where: String = "",
    val app: String = "",
    val state: String = "",
    val kind: String? = null,
    val msg: String? = null,
    val turn: Long = 0,
    val title: String = "",
    val cols: Int = 0,
    val rows: Int = 0,
) {
    val blocked
        get() = state == "blocked"

    val label
        get() = tab.ifEmpty { where.ifEmpty { address } }
}

@Serializable
data class Space(val id: String, val name: String, val agents: List<Agent> = emptyList())

@Serializable
data class TranscriptItem(val role: String, val kind: String, val text: String, val ts: Long = 0)

@Serializable
data class TranscriptPage(
    val source: String,
    val hint: String? = null,
    val model: String = "",
    val items: List<TranscriptItem> = emptyList(),
    @SerialName("next_cursor") val nextCursor: String = "",
)

/**
 * One key of a question: a digit for an option, or "tab" for Next after a multi-select. [detail] is
 * the description under the option.
 */
@Serializable data class PromptOption(val key: String, val label: String, val detail: String = "")

@Serializable
data class PromptView(
    val app: String = "",
    val kind: String? = null,
    val msg: String = "",
    val block: String = "",
    /** A question's block above its options: the options show as keys, not twice. */
    val question: String = "",
    val options: List<PromptOption> = emptyList(),
    val turn: Long = 0,
    @SerialName("prompt_hash") val promptHash: String = "",
    val actionable: Boolean = false,
    val reason: String? = null,
) {
    /** A permission prompt that Approve and Deny answer. A question has [options] instead. */
    val approvable
        get() = actionable && options.isEmpty()
}

@Serializable data class Skills(val skills: List<String> = emptyList())

@Serializable
data class Hello(
    @SerialName("daemon_version") val version: String = "",
    @SerialName("server_time_ms") val serverTimeMs: Long = 0,
    val endpoints: List<String> = emptyList(),
)

@Serializable data class ResultBody(val result: String)

@Serializable
data class ErrorBody(
    val error: String = "",
    val reason: String? = null,
    @SerialName("idle_ms") val idleMs: Long? = null,
)

/** [decision] is approve, deny or answer. An answer names the option's [key]. */
@Serializable
data class DecisionRequest(
    @SerialName("request_id") val requestId: String,
    val decision: String,
    val turn: Long,
    @SerialName("prompt_hash") val promptHash: String,
    val key: String? = null,
)

@Serializable
data class MessageRequest(@SerialName("request_id") val requestId: String, val text: String)

@Serializable data class EnrollRequest(val token: String, val name: String)

@Serializable data class EnrollResponse(@SerialName("device_id") val deviceId: String)
