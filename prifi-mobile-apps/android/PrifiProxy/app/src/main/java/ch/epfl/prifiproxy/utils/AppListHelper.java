package ch.epfl.prifiproxy.utils;

import android.content.Context;
import android.content.SharedPreferences;
import android.content.pm.ApplicationInfo;
import android.content.pm.PackageManager;

import java.text.Collator;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.Collections;
import java.util.HashSet;
import java.util.List;
import java.util.Locale;
import java.util.Set;

import ch.epfl.prifiproxy.R;

import static android.content.pm.PackageManager.GET_META_DATA;

public class AppListHelper {
    public enum Sort {LABEL, PACKAGE_NAME}

    public enum Order {ASC, DESC}

    public static String APP_LIST_KEY = "appList";

    // Apps that may be installed by the system, but are commonly used by the user

    private static final Set<String> commonSystemApps;

    static {
        List<String> apps = Arrays.asList(
                "com.android.chrome",
                "com.facebook.katana",
                "com.facebook.orca",
                "com.google.android.youtube",
                "com.skype.raider",
                "com.twitter.android",
                "com.whatapp",
                "com.instagram.android"
        );
        commonSystemApps = new HashSet<>(apps);
    }

    public static List<AppInfo> getApps(Context context, Sort sort, boolean descending,
                                        boolean showSystemPackages) {
        List<AppInfo> apps = new ArrayList<>();
        PackageManager packageManager = context.getPackageManager();
        List<ApplicationInfo> installedApps = packageManager.getInstalledApplications(GET_META_DATA);

        Set<String> prifiApps = getPrifiApps(context);

        for (ApplicationInfo applicationInfo : installedApps) {
            if (!showSystemPackages
                    && !isCommonSystemApp(applicationInfo.packageName)
                    && isSystemPackage(applicationInfo))
                continue;

            if (!hasInternet(context, applicationInfo.packageName))
                continue;

            AppInfo appInfo = new AppInfo();
            appInfo.name = applicationInfo.name;
            appInfo.label = (String) applicationInfo.loadLabel(packageManager);
            appInfo.packageName = applicationInfo.packageName;
            appInfo.icon = applicationInfo.icon;
            appInfo.usePrifi = prifiApps.contains(appInfo.packageName);
            apps.add(appInfo);
        }

        final Collator collator = Collator.getInstance(Locale.getDefault());
        collator.setStrength(Collator.SECONDARY);

        switch (sort) {
            case LABEL:
                Collections.sort(apps, (a, b) -> collator.compare(a.label, b.label));
                break;
            case PACKAGE_NAME:
                Collections.sort(apps, (a, b) -> collator.compare(a.packageName, b.packageName));
                break;
        }

        if (descending) {
            Collections.reverse(apps);
        }

        return apps;
    }

    /**
     * Get list of apps that use prifi
     */
    public static Set<String> getPrifiApps(Context context) {
        SharedPreferences prefs = context.getSharedPreferences(context.getString(R.string.prifi_config_shared_preferences), Context.MODE_PRIVATE);
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
        if (old.equals(newApps)) {
            return;
        }

        SharedPreferences prefs = context.getSharedPreferences(context.getString(R.string.prifi_config_shared_preferences), Context.MODE_PRIVATE);
        SharedPreferences.Editor editor = prefs.edit();

        editor.putStringSet(APP_LIST_KEY, newApps);
        editor.apply();
    }

    private static boolean isCommonSystemApp(String packageName) {
        return commonSystemApps.contains(packageName);
    }

    private static boolean isSystemPackage(ApplicationInfo applicationInfo) {
        return ((applicationInfo.flags & ApplicationInfo.FLAG_SYSTEM) != 0);
    }

    private static boolean hasInternet(Context context, String packageName) {
        PackageManager pm = context.getPackageManager();
        int permission = pm.checkPermission("android.permission.INTERNET", packageName);
        return permission == PackageManager.PERMISSION_GRANTED;
    }
}
