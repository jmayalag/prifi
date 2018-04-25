package ch.epfl.prifiproxy.activities;

import android.content.BroadcastReceiver;
import android.content.Context;
import android.content.Intent;
import android.content.IntentFilter;
import android.os.Build;
import android.os.HandlerThread;
import android.os.Process;
import android.support.v4.app.NavUtils;
import android.support.v7.app.AppCompatActivity;
import android.os.Bundle;
import android.view.MenuItem;
import android.view.View;
import android.widget.ScrollView;
import android.widget.TextView;

import java.util.Objects;

import ch.epfl.prifiproxy.R;
import ch.epfl.prifiproxy.utils.OnScreenLogHandler;

/**
 * The activity that displays the logcat content.
 */
public class OnScreenLogActivity extends AppCompatActivity {

    private final String ON_SCREEN_LOG_THREAD_NAME = "ch.epfl.prifiproxy.ON_SCREEN_LOG_THREAD";

    private ScrollView mScrollView;
    private TextView onScreenLogTextView;

    private BroadcastReceiver mBroadcastReceiver;

    private HandlerThread mHandlerThread;
    private OnScreenLogHandler mOnScreenLogHandler;

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setContentView(R.layout.activity_on_screen_log);

        Objects.requireNonNull(getSupportActionBar()).setDisplayHomeAsUpEnabled(true);

        // Views
        mScrollView = findViewById(R.id.logScrollView);
        onScreenLogTextView = findViewById(R.id.logTextView);

        // Broadcast Receiver
        mBroadcastReceiver = new BroadcastReceiver() {
            @Override
            public void onReceive(Context context, Intent intent) {
                String action = intent.getAction();

                if (action != null) {
                    switch (action) {
                        case OnScreenLogHandler.UPDATE_LOG_BROADCAST_ACTION:
                            String log = intent.getExtras().getString(OnScreenLogHandler.UPDATE_LOG_INTENT_KEY);
                            if (log != null) {
                                updateOnScreenLog(log);
                            }
                            break;

                        default:
                            break;
                    }
                }
            }
        };

        // Handler and HandlerThread
        mHandlerThread = new HandlerThread(ON_SCREEN_LOG_THREAD_NAME, Process.THREAD_PRIORITY_BACKGROUND);
        mHandlerThread.start();
        mOnScreenLogHandler = new OnScreenLogHandler(mHandlerThread.getLooper());
    }

    @Override
    protected void onResume() {
        super.onResume();

        // Register Receiver
        IntentFilter intentFilter = new IntentFilter();
        intentFilter.addAction(OnScreenLogHandler.UPDATE_LOG_BROADCAST_ACTION);
        registerReceiver(mBroadcastReceiver, intentFilter);

        // Trigger Handler
        mOnScreenLogHandler.sendEmptyMessage(OnScreenLogHandler.UPDATE_LOG_MESSAGE_WHAT);
    }

    @Override
    protected void onPause() {
        super.onPause();

        // Unregister Receiver
        unregisterReceiver(mBroadcastReceiver);

        // Quit Handler Thread and Remove messages from the queue
        mOnScreenLogHandler.removeMessages(OnScreenLogHandler.UPDATE_LOG_MESSAGE_WHAT);
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.JELLY_BEAN_MR2) {
            mHandlerThread.quitSafely();
        } else {
            mHandlerThread.quit();
        }

        // Clean the view
        updateOnScreenLog("");
    }

    @Override
    public boolean onOptionsItemSelected(MenuItem item) {
        switch (item.getItemId()) {
            // Action Bar back button
            case android.R.id.home:
                NavUtils.navigateUpFromSameTask(this);
                return true;
        }
        return super.onOptionsItemSelected(item);
    }

    /**
     * Update the text view with a given string
     * @param s logcat content
     */
    private void updateOnScreenLog(String s) {
        onScreenLogTextView.setText(s);
        mScrollView.post(() -> mScrollView.fullScroll(View.FOCUS_DOWN));
    }

}
