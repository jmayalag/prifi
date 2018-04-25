package ch.epfl.prifiproxy.utils;

import android.content.Intent;
import android.os.Handler;
import android.os.Looper;
import android.os.Message;
import android.util.Log;

import ch.epfl.prifiproxy.PrifiProxy;

import java.io.BufferedReader;
import java.io.IOException;
import java.io.InputStreamReader;

/**
 * This class allows us to display Logcat content (logs) directly in the app.
 */
public class OnScreenLogHandler extends Handler {

    public static final String UPDATE_LOG_BROADCAST_ACTION = "ch.epfl.prifiproxy.UPDATE_LOG_BROADCAST_ACTION"; // Broadcast Action Keyword
    public static final String UPDATE_LOG_INTENT_KEY = "ch.epfl.prifiproxy.LOG"; // Intent Action Keyword
    public static final int UPDATE_LOG_MESSAGE_WHAT = 1;

    private final int DELAY = 2000;

    public OnScreenLogHandler(Looper looper) {
       super(looper);
    }

    /**
     * The handler will broadcast update commands with logcat content every DELAY seconds.
     * The UI part must implement a corresponding broadcast receiver.
     * @param msg Handler's message
     */
    @Override
    public void handleMessage(Message msg) {
        if (msg.what == UPDATE_LOG_MESSAGE_WHAT) {
            String log = getLogCatContent();
            Intent intent = new Intent(UPDATE_LOG_BROADCAST_ACTION);
            intent.putExtra(UPDATE_LOG_INTENT_KEY, log);

            PrifiProxy.getContext().sendBroadcast(intent);
            this.sendEmptyMessageDelayed(UPDATE_LOG_MESSAGE_WHAT, DELAY);
        }
    }

    /**
     * This function will execute "adb logcat -d -s TAG".
     * @return logcat content
     */
    private String getLogCatContent() {
        StringBuilder logBuilder = new StringBuilder();

        try {
            Process process = Runtime.getRuntime().exec("logcat -d -s GoLog");
            BufferedReader bufferedReader = new BufferedReader(new InputStreamReader(process.getInputStream()));

            String line;
            while ((line = bufferedReader.readLine()) != null) {
                logBuilder.append(line);
            }
        } catch (IOException e) {

        }

        return logBuilder.toString();
    }

}
