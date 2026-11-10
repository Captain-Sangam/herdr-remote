package dev.herdrremote

import android.app.Application
import androidx.lifecycle.ProcessLifecycleOwner
import dev.herdrremote.net.DeviceKey
import dev.herdrremote.net.HerdrApi
import dev.herdrremote.net.Store
import dev.herdrremote.watch.AgentsRepository
import dev.herdrremote.watch.WatchService

/** Process-wide objects. No DI framework (ADR 0015). */
class HerdrRemoteApp : Application() {
    lateinit var store: Store
    @Volatile private var apiCache: HerdrApi? = null
    val repository by lazy { AgentsRepository(this) }

    override fun onCreate() {
        super.onCreate()
        store = Store(this)
        instance = this
        ProcessLifecycleOwner.get().lifecycle.addObserver(Gate.Observer())
    }

    /** The API for the paired Mac, or null when not paired. */
    fun api(): HerdrApi? {
        val pin = store.pin ?: return null
        if (store.deviceId == null) return null
        return apiCache ?: HerdrApi(store, pin, DeviceKey.ensure()).also { apiCache = it }
    }

    fun forgetMac() {
        WatchService.stop(this)
        apiCache = null
        repository.stop()
        store.forget()
    }

    companion object {
        lateinit var instance: HerdrRemoteApp
            private set
    }
}
