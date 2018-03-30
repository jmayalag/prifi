package com.example.junchen.prifiproxy.activities;

import android.app.ActivityManager;
import android.app.ProgressDialog;
import android.content.BroadcastReceiver;
import android.content.Context;
import android.content.Intent;
import android.content.IntentFilter;
import android.os.Bundle;
import android.os.HandlerThread;
import android.os.Process;
import android.support.v7.app.AppCompatActivity;
import android.view.View;
import android.widget.Button;
import android.widget.ScrollView;
import android.widget.Switch;
import android.widget.TextView;

import com.example.junchen.prifiproxy.R;
import com.example.junchen.prifiproxy.services.PrifiService;
import com.example.junchen.prifiproxy.utils.OnScreenLogHandler;

import java.util.concurrent.atomic.AtomicBoolean;

import prifiMobile.PrifiMobile;

public class MainActivity extends AppCompatActivity {

    private final String ON_SCREEN_LOG_THREAD = "ON_SCREEN_LOG";
    private final String EMPTY_TEXT_VIEW = "";

    private AtomicBoolean isPrifiServiceRunning;
    private Button startButton;
    private Button stopButton;
    private Switch logSwitch;
    private ScrollView mScrollView;
    private TextView onScreenLogTextView;
    private ProgressDialog mProgessDialog;

    private BroadcastReceiver mBroadcastReceiver;

    private HandlerThread mHandlerThread;
    private OnScreenLogHandler mOnScreenLogHandler;

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setContentView(R.layout.activity_main);

        startButton = findViewById(R.id.startButton);
        stopButton = findViewById(R.id.stopButton);
        logSwitch = findViewById(R.id.logSwitch);
        mScrollView = findViewById(R.id.mainScrollView);
        onScreenLogTextView = findViewById(R.id.logTextView);

        mBroadcastReceiver = new BroadcastReceiver() {
            @Override
            public void onReceive(Context context, Intent intent) {
                String action = intent.getAction();

                switch (action) {
                    case PrifiService.PRIFI_STOPPED_BROADCAST_ACTION:
                        if (mProgessDialog.isShowing()) {
                            mProgessDialog.dismiss();
                        }
                        break;

                    case OnScreenLogHandler.UPDATE_LOG_BROADCAST_ACTION:
                        String log = intent.getExtras().getString(OnScreenLogHandler.UPDATE_LOG_INTENT_KEY);
                        updateOnScreenLog(log);
                        break;

                    default:
                        break;
                }

            }
        };

        mHandlerThread = new HandlerThread(ON_SCREEN_LOG_THREAD, Process.THREAD_PRIORITY_BACKGROUND);
        mHandlerThread.start();
        mOnScreenLogHandler = new OnScreenLogHandler(mHandlerThread.getLooper());

        startButton.setOnClickListener(view -> startPrifiService());

        stopButton.setOnClickListener(view -> stopPrifiService());

        logSwitch.setOnCheckedChangeListener((compoundButton, isChecked) -> {
            if (isChecked) {
                mOnScreenLogHandler.sendEmptyMessage(OnScreenLogHandler.UPDATE_LOG_MESSAGE_WHAT);
            } else {
                mOnScreenLogHandler.removeMessages(OnScreenLogHandler.UPDATE_LOG_MESSAGE_WHAT);
                updateOnScreenLog(EMPTY_TEXT_VIEW);
            }
        });
    }

    @Override
    protected void onResume() {
        super.onResume();

        isPrifiServiceRunning = new AtomicBoolean(isMyServiceRunning(PrifiService.class));

        IntentFilter intentFilter = new IntentFilter();
        intentFilter.addAction(PrifiService.PRIFI_STOPPED_BROADCAST_ACTION);
        intentFilter.addAction(OnScreenLogHandler.UPDATE_LOG_BROADCAST_ACTION);
        registerReceiver(mBroadcastReceiver, intentFilter);
    }

    @Override
    protected void onPause() {
        super.onPause();
        unregisterReceiver(mBroadcastReceiver);
    }

    @Override
    protected void onRestart() {
        super.onRestart();
        if (logSwitch.isChecked()) {
            mOnScreenLogHandler.sendEmptyMessage(OnScreenLogHandler.UPDATE_LOG_MESSAGE_WHAT);
        }
    }

    @Override
    protected void onStop() {
        super.onStop();
        mOnScreenLogHandler.removeMessages(OnScreenLogHandler.UPDATE_LOG_MESSAGE_WHAT);
    }

    private void updateOnScreenLog(String s) {
        onScreenLogTextView.setText(s);
        mScrollView.post(() -> mScrollView.fullScroll(View.FOCUS_DOWN));
    }

    private void startPrifiService() {
        if (isPrifiServiceRunning.compareAndSet(false, true)) {
            startService(new Intent(this, PrifiService.class));
        }
    }

    private void stopPrifiService() {
        if (isPrifiServiceRunning.compareAndSet(true, false)) {
            mProgessDialog = ProgressDialog.show(this, "Stopping PriFi", "Please wait");
            PrifiMobile.stopClient(); // StopClient will make the service to shutdown by itself
        }
    }

    private boolean isMyServiceRunning(Class<?> serviceClass) {
        ActivityManager manager = (ActivityManager) getSystemService(Context.ACTIVITY_SERVICE);
        for (ActivityManager.RunningServiceInfo service : manager.getRunningServices(Integer.MAX_VALUE)) {
            if (serviceClass.getName().equals(service.service.getClassName())) {
                return true;
            }
        }
        return false;
    }

}