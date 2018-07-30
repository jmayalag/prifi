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
    private static String APP_LIST_KEY = "appList";

    public static List<AppInfo> getApps(Context context) {
        List<AppInfo> apps = new ArrayList<>();
        PackageManager packageManager = context.getPackageManager();
        List<ApplicationInfo> installedApps = packageManager.getInstalledApplications(GET_META_DATA);

        Collections.sort(installedApps, (o1, o2) -> o1.packageName.compareTo(o2.packageName));

        Set<String> prifiApps = getPrifiApps(context);

        for (ApplicationInfo applicationInfo : installedApps) {
            if (isSystemPackage(applicationInfo))
                continue;

            AppInfo appInfo = new AppInfo();
            appInfo.name = applicationInfo.name;
            appInfo.label = (String) applicationInfo.loadLabel(packageManager);
            appInfo.packageName = applicationInfo.packageName;
            appInfo.icon = applicationInfo.icon;
            appInfo.usePrifi = prifiApps.contains(appInfo.packageName);
            apps.add(appInfo);
        }

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
        SharedPreferences prefs = PreferenceManager.getDefaultSharedPreferences(context);
        SharedPreferences.Editor editor = prefs.edit();

        Set<String> apps = new HashSet<>(packageNames);
        editor.putStringSet(APP_LIST_KEY, apps);
        editor.apply();
    }

    private static boolean isSystemPackage(ApplicationInfo applicationInfo) {
        return ((applicationInfo.flags & ApplicationInfo.FLAG_SYSTEM) != 0);
    }
}
