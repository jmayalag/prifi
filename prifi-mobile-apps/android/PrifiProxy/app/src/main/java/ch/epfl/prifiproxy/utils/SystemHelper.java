package ch.epfl.prifiproxy.utils;

import android.app.ActivityManager;
import android.content.Context;
import android.content.pm.PackageManager;

/**
 * This class provides functions that interact with OS.
 */
public class SystemHelper {

    /**
     * Check if the given service is running or not.
     * @param serviceClass Service
     * @return true if running, otherwise false
     */
    public static boolean isMyServiceRunning(Class<?> serviceClass, Context context) {
        ActivityManager manager = (ActivityManager) context.getSystemService(Context.ACTIVITY_SERVICE);
        if (manager != null) {
            for (ActivityManager.RunningServiceInfo service : manager.getRunningServices(Integer.MAX_VALUE)) {
                if (serviceClass.getName().equals(service.service.getClassName())) {
                    return true;
                }
            }
        }
        return false;
    }

    /**
     * Check if the given app is installed or not.
     * @param context context
     * @param appName app package name
     * @return true if installed, otherwise false
     */
    public static boolean isAppAvailable(Context context, String appName) {
        PackageManager packageManager = context.getPackageManager();
        try {
            packageManager.getPackageInfo(appName, PackageManager.GET_ACTIVITIES);
            return true;
        } catch (PackageManager.NameNotFoundException e) {
            return false;
        }
    }

}
