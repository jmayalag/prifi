package ch.epfl.prifiproxy.services;

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

import ch.epfl.prifiproxy.R;
import ch.epfl.prifiproxy.activities.MainActivity;
import prifiMobile.PrifiMobile;

/**
 * This class controls the PriFi Core as Android Service (Foreground Service).
 *
 * PriFi Service Lifecycle:
 *
 * 1. onStartCommand triggered by user action
 *
 * 2. Handler's handleMessage is triggered by the previous action
 *
 * 3. The blocking PriFi Core runs on a separate thread
 *
 * 4. The PriFi Core will return when a stopped signal is given (check Golang code).
 *    The finally block will be executed and the service shuts down by itself.
 *
 * 5. While shutting down the service (onDestroy), a broadcast is sent in order to update UI.
 */
public class PrifiService extends Service {

    public static final String PRIFI_STOPPED_BROADCAST_ACTION = "ch.epfl.prifiproxy.PRIFI_STOPPED_BROADCAST_ACTION"; // Broadcast Action Key

    private static final String PRIFI_SERVICE_THREAD_NAME = "ch.epfl.prifiproxy.PRIFI_SERVICE_THREAD";
    private static final String PRIFI_SERVICE_NOTIFICATION_CHANNEL = "ch.epfl.prifiproxy.PRIFI_SERVICE_NOTIFICATION_CHANNEL";
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
            try {
                PrifiMobile.startClient(); // startClient is a blocking method.
            } finally {
                stopForeground(true);
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
        Toast.makeText(this, getString(R.string.prifi_service_starting), Toast.LENGTH_SHORT).show();

        Message msg = mServiceHandler.obtainMessage();
        msg.arg1 = startId;
        mServiceHandler.sendMessage(msg); // Trigger handlerMessage

        return START_NOT_STICKY;
    }

    @Override
    public IBinder onBind(Intent intent) {
        return null; // onBind is not supported
    }

    @Override
    public void onDestroy() {
        // While destroying the service, a broadcast is sent to update UI.
        sendBroadcast(new Intent(PRIFI_STOPPED_BROADCAST_ACTION));
        Toast.makeText(this, getString(R.string.prifi_service_stopped), Toast.LENGTH_SHORT).show();
        mServiceThread.quit();
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
