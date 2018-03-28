package com.example.junchen.prifiproxy.services;

import android.app.Notification;
import android.app.PendingIntent;
import android.app.Service;
import android.content.Intent;
import android.os.Handler;
import android.os.HandlerThread;
import android.os.IBinder;
import android.os.Looper;
import android.os.Message;
import android.os.Process;
import android.support.v4.app.NotificationCompat;
import android.widget.Toast;

import com.example.junchen.prifiproxy.R;
import com.example.junchen.prifiproxy.activities.MainActivity;

import prifiMobile.PrifiMobile;

/**
 * Created by junchen on 23.03.18.
 */

public class PrifiService extends Service {

    private static final String PRIFI_SERVICE_THREAD_NAME = "PrifiService";
    private static final String PRIFI_SERVICE_NOTIFICATION_CHANNEL = "PrifiChannel";
    private static final int PRIFI_SERVICE_NOTIFICATION_ID = 42;

    private Looper mServiceLooper;
    private ServiceHandler mServiceHandler;
    private HandlerThread mServiceThread;

    // Handler that receives messages from the thread
    private final class ServiceHandler extends Handler {
        public ServiceHandler(Looper looper) {
            super(looper);
        }

        @Override
        public void handleMessage(Message msg) {
            // Normally we would do some work here, like download a file.
            // For our sample, we just sleep for 5 seconds.
            try {
                PrifiMobile.startClient();
            } finally {
                stopSelf(msg.arg1);
            }
        }
    }

    @Override
    public void onCreate() {
        mServiceThread = new HandlerThread(PRIFI_SERVICE_THREAD_NAME, Process.THREAD_PRIORITY_BACKGROUND);
        mServiceThread.start();

        mServiceLooper = mServiceThread.getLooper();
        mServiceHandler = new ServiceHandler(mServiceLooper);

        Notification notification = constructForegroundNotification();
        startForeground(PRIFI_SERVICE_NOTIFICATION_ID, notification);
    }

    @Override
    public int onStartCommand(Intent intent, int flags, int startId) {
        Toast.makeText(this, "Service starting", Toast.LENGTH_SHORT).show();

        // For each start request, send a message to start a job and deliver the
        // start ID so we know which request we're stopping when we finish the job
        Message msg = mServiceHandler.obtainMessage();
        msg.arg1 = startId;
        mServiceHandler.sendMessage(msg);

        // If we get killed, after returning from here, restart
        return START_NOT_STICKY;
    }

    @Override
    public IBinder onBind(Intent intent) {
        // We don't provide binding, so return null
        return null;
    }

    @Override
    public void onDestroy() {
        Toast.makeText(this, "Service stopped", Toast.LENGTH_SHORT).show();
        mServiceThread.quit();

        mServiceThread = null;
        mServiceHandler = null;
    }

    private Notification constructForegroundNotification() {
        Intent notificationIntent = new Intent(this, MainActivity.class);
        PendingIntent pendingIntent =
                PendingIntent.getActivity(this, 0, notificationIntent, 0);

        Notification notification =
                new NotificationCompat.Builder(this, PRIFI_SERVICE_NOTIFICATION_CHANNEL)
                        .setSmallIcon(R.mipmap.ic_launcher)
                        .setContentTitle(getText(R.string.prifi_service_notification_title))
                        .setContentText(getText(R.string.prifi_service_notification_message))
                        .setContentIntent(pendingIntent)
                        .build();

        return notification;
    }

}
