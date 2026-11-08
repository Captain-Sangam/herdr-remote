package dev.herdrremote.ui

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import dev.herdrremote.Gate
import dev.herdrremote.HerdrRemoteApp
import dev.herdrremote.net.Agent
import dev.herdrremote.net.ApiException
import dev.herdrremote.net.DecisionRequest
import dev.herdrremote.net.MessageRequest
import dev.herdrremote.net.PromptView
import dev.herdrremote.net.TranscriptItem
import dev.herdrremote.net.UnreachableException
import dev.herdrremote.net.Uuid7
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.flow.stateIn
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch

data class TranscriptState(
    val source: String = "",
    val hint: String? = null,
    val items: List<TranscriptItem> = emptyList(),
    val nextCursor: String = "",
    val loading: Boolean = false,
    val error: String? = null,
)

/** What the phone says for each result and each 409 reason (steps 7 and 11). */
object Texts {
    fun result(result: String): String = when (result) {
        "applied" -> "Done. The agent continues."
        "sent_unconfirmed" -> "Sent. The agent has not confirmed it yet."
        "uncertain" -> "Someone used the Mac at the same moment. Check the transcript before you act again."
        "sent" -> "Sent to the agent."
        "outcome_unknown" -> "Outcome unknown. Check the transcript before you send again."
        else -> result
    }

    fun error(e: Throwable): String = when (e) {
        is ApiException -> when (e.reason) {
            "stale" -> "The prompt has moved on. Reload it."
            "changed" -> "The prompt changed. Reload it."
            "already_answered" -> "This prompt was already answered."
            "mac_in_use" -> "Someone is using the Mac right now (idle ${((e.idleMs ?: 0) / 1000)} s). Try again when it is free."
            "not_actionable" -> "This prompt cannot be answered from the phone. Answer it on the Mac."
            "blocked" -> "The agent is waiting on a prompt. Answer that first."
            "not_agent" -> "This terminal is not an agent."
            "expired_request" -> "The phone's clock differs from the Mac's. Try again."
            "cursor" -> "The transcript changed. Start again from the end."
            else -> when (e.status) {
                429 -> "Too fast. Wait a second and try again."
                503 -> "The Mac cannot reach Herdr right now."
                else -> e.error
            }
        }
        is UnreachableException -> "The Mac is not reachable."
        else -> "Outcome unknown. Check the transcript before you send again."
    }

    /** After an action: the daemon's own reply keeps its reason; any other failure may have reached the Mac. */
    fun actionError(e: Throwable): String = if (e is ApiException) error(e) else result("outcome_unknown")
}

class AgentViewModel(private val app: HerdrRemoteApp, val address: String) : ViewModel() {
    val agent: StateFlow<Agent?> = app.repository.state.map { it.agents[address] }.stateIn(viewModelScope, SharingStarted.Eagerly, app.repository.state.value.agents[address])

    private val _transcript = MutableStateFlow(TranscriptState())
    val transcript = _transcript.asStateFlow()
    private val _prompt = MutableStateFlow<PromptView?>(null)
    val prompt = _prompt.asStateFlow()
    private val _note = MutableStateFlow<String?>(null)
    /** The last action's outcome, in words. */
    val note = _note.asStateFlow()
    private val _busy = MutableStateFlow(false)
    val busy = _busy.asStateFlow()

    private var lastTurn: Long = -1

    init {
        loadTranscript()
        viewModelScope.launch {
            agent.collect { a ->
                if (a != null && (a.turn != lastTurn)) {
                    lastTurn = a.turn
                    loadTranscript()
                }
                if (a?.blocked == true) refreshPrompt() else _prompt.value = null
            }
        }
    }

    fun loadTranscript() {
        viewModelScope.launch {
            _transcript.update { it.copy(loading = true, error = null) }
            try {
                val page = app.api()!!.transcript(address, null)
                _transcript.value = TranscriptState(page.source, page.hint, page.items, page.nextCursor)
            } catch (e: Exception) {
                _transcript.update { it.copy(loading = false, error = Texts.error(e)) }
            }
        }
    }

    /** Pages back: the earlier items go in front. A stale cursor starts again from the end. */
    fun loadOlder() {
        val cur = _transcript.value
        if (cur.loading || cur.nextCursor.isEmpty()) return
        viewModelScope.launch {
            _transcript.update { it.copy(loading = true) }
            try {
                val page = app.api()!!.transcript(address, cur.nextCursor)
                _transcript.update { it.copy(items = page.items + it.items, nextCursor = page.nextCursor, loading = false) }
            } catch (e: ApiException) {
                if (e.reason == "cursor") loadTranscript() else _transcript.update { it.copy(loading = false, error = Texts.error(e)) }
            } catch (e: Exception) {
                _transcript.update { it.copy(loading = false, error = Texts.error(e)) }
            }
        }
    }

    fun refreshPrompt() {
        viewModelScope.launch {
            try { _prompt.value = app.api()!!.prompt(address) } catch (e: Exception) { _note.value = Texts.error(e) }
        }
    }

    /** Approve or deny. Never retried by the app: an unknown outcome is shown as such. */
    fun decide(approve: Boolean) {
        val p = _prompt.value ?: return
        if (Gate.locked) { _note.value = "Unlock the app first."; return }
        if (!p.actionable) { _note.value = "Answer this on the Mac."; return }
        val api = app.api() ?: return
        _busy.value = true
        viewModelScope.launch {
            try {
                val res = api.decide(address, DecisionRequest(Uuid7.make(api.now()), if (approve) "approve" else "deny", p.turn, p.promptHash))
                _note.value = Texts.result(res)
            } catch (e: Exception) {
                _note.value = Texts.actionError(e)
            }
            _busy.value = false
            refreshPrompt()
        }
    }

    fun send(text: String, onSent: () -> Unit) {
        if (Gate.locked) { _note.value = "Unlock the app first."; return }
        val api = app.api() ?: return
        if (text.isBlank()) return
        _busy.value = true
        viewModelScope.launch {
            try {
                _note.value = Texts.result(api.message(address, MessageRequest(Uuid7.make(api.now()), text)))
                onSent()
            } catch (e: Exception) {
                _note.value = Texts.actionError(e)
            }
            _busy.value = false
        }
    }
}
