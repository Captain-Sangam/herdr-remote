package dev.herdrremote.net

import android.content.Context
import android.content.SharedPreferences

/** What the app remembers about its Mac. Nothing here is a secret; the key lives in the Keystore. */
class Store(context: Context) {
    private val p: SharedPreferences = context.getSharedPreferences("herdr-remote", Context.MODE_PRIVATE)

    var pin: String?
        get() = p.getString("pin", null)
        set(v) = p.edit().putString("pin", v).apply()
    var endpoints: List<String>
        get() = p.getString("endpoints", "").orEmpty().split(',').filter { it.isNotEmpty() }
        set(v) = p.edit().putString("endpoints", v.joinToString(",")).apply()
    var lastGood: String?
        get() = p.getString("last_good", null)
        set(v) = p.edit().putString("last_good", v).apply()
    var port: Int
        get() = p.getInt("port", 7766)
        set(v) = p.edit().putInt("port", v).apply()
    var deviceId: String?
        get() = p.getString("device_id", null)
        set(v) = p.edit().putString("device_id", v).apply()
    var macName: String
        get() = p.getString("mac_name", "").orEmpty()
        set(v) = p.edit().putString("mac_name", v).apply()
    var watch: Boolean
        get() = p.getBoolean("watch", false)
        set(v) = p.edit().putBoolean("watch", v).apply()
    /** server_time_ms minus the phone clock, from the last /v1/hello. */
    var clockOffsetMs: Long
        get() = p.getLong("clock_offset", 0)
        set(v) = p.edit().putLong("clock_offset", v).apply()

    val paired: Boolean get() = pin != null && deviceId != null

    fun forget() {
        p.edit().clear().apply()
        DeviceKey.forget()
    }
}
