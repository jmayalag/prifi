package ch.epfl.prifiproxy.utils;

import android.content.Context;
import android.content.SharedPreferences;
import android.content.pm.ApplicationInfo;
import android.content.pm.PackageManager;
import android.preference.PreferenceManager;

import java.util.ArrayList;
import java.util.Collections;
import java.util.HashSet;
import java.util.List;
import java.util.Set;

import static android.content.pm.PackageManager.GET_META_DATA;

public class AppListHelper {
    public static String APP_LIST_KEY = "appList";

    public static List<AppInfo> getApps(Context context) {
        List<AppInfo> apps = new ArrayList<>();
        PackageManager packageManager = context.getPackageManager();
        List<ApplicationInfo> installedApps = packageManager.getInstalledApplications(GET_META_DATA);

        Set<String> prifiApps = getPrifiApps(context);

        for (ApplicationInfo applicationInfo : installedApps) {
//            if (isSystemPackage(applicationInfo))
//                continue;

            AppInfo appInfo = new AppInfo();
            appInfo.name = applicationInfo.name;
            appInfo.label = (String) applicationInfo.loadLabel(packageManager);
            appInfo.packageName = applicationInfo.packageName;
            appInfo.icon = applicationInfo.icon;
            appInfo.usePrifi = prifiApps.contains(appInfo.packageName);
            apps.add(appInfo);
        }

        Collections.sort(apps, (o1, o2) -> o1.label.compareTo(o2.label));

        return apps;
    }

    /**
     * Get list of apps that use prifi
     */
    public static Set<String> getPrifiApps(Context context) {
        SharedPreferences prefs = PreferenceManager.getDefaultSharedPreferences(context);
        Set<String> apps = prefs.getStringSet(APP_LIST_KEY, new HashSet<>());
        apps = new HashSet<>(apps);
        return apps;
    }

    /**
     * Saves the list of apps that use prifi.
     */
    public static void savePrifiApps(Context context, List<String> packageNames) {
        Set<String> old = getPrifiApps(context);
        Set<String> newApps = new HashSet<>(packageNames);
        // Save only if there were changes
        if (!old.equals(newApps)) {
            return;
        }

        SharedPreferences prefs = PreferenceManager.getDefaultSharedPreferences(context);
        SharedPreferences.Editor editor = prefs.edit();

        editor.putStringSet(APP_LIST_KEY, newApps);
        editor.apply();
    }

    private static boolean isSystemPackage(ApplicationInfo applicationInfo) {
        return ((applicationInfo.flags & ApplicationInfo.FLAG_SYSTEM) != 0);
    }
}
