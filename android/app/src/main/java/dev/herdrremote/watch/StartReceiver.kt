package dev.herdrremote.watch

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import dev.herdrremote.net.Store

/** Starts watching again after a reboot or an app update, if the toggle is on. */
class StartReceiver : BroadcastReceiver() {
    override fun onReceive(context: Context, intent: Intent) {
        if (intent.action != Intent.ACTION_BOOT_COMPLETED && intent.action != Intent.ACTION_MY_PACKAGE_REPLACED) return
        val store = Store(context)
        if (store.paired && store.watch) WatchService.start(context)
    }
}
