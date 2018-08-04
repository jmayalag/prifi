package ch.epfl.prifiproxy.activities;

import android.content.SharedPreferences;
import android.graphics.PorterDuff;
import android.graphics.drawable.Drawable;
import android.os.AsyncTask;
import android.os.Bundle;
import android.support.v7.app.AppCompatActivity;
import android.support.v7.widget.LinearLayoutManager;
import android.support.v7.widget.RecyclerView;
import android.support.v7.widget.SearchView;
import android.support.v7.widget.Toolbar;
import android.util.Log;
import android.view.Menu;
import android.view.MenuItem;

import java.lang.ref.WeakReference;
import java.util.ArrayList;
import java.util.List;

import ch.epfl.prifiproxy.R;
import ch.epfl.prifiproxy.adapters.AppSelectionAdapter;
import ch.epfl.prifiproxy.listeners.OnAppCheckedListener;
import ch.epfl.prifiproxy.utils.AppInfo;
import ch.epfl.prifiproxy.utils.AppListHelper;

public class AppSelectionActivity extends AppCompatActivity implements OnAppCheckedListener, SearchView.OnQueryTextListener {
    private static final String TAG = "PRIFI_APP_SELECTION";
    private RecyclerView mRecyclerView;
    private AppSelectionAdapter mAdapter;
    private RecyclerView.LayoutManager mLayoutManager;
    private List<AppInfo> mAppList;
    private SearchView searchView;

    // Preferences
    private AppListHelper.Sort sortField;
    private boolean sortDescending;
    private boolean showSystemApps;

    private UpdateAppListTask currentUpdate;

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setContentView(R.layout.activity_app_selection);
        Toolbar toolbar = findViewById(R.id.toolbar);
        setSupportActionBar(toolbar);

        getSupportActionBar().setDisplayHomeAsUpEnabled(true);

        getPreferences();

        mRecyclerView = findViewById(R.id.app_list);
        mRecyclerView.setHasFixedSize(true);

        mLayoutManager = new LinearLayoutManager(this);
        mRecyclerView.setLayoutManager(mLayoutManager);

        mAppList = new ArrayList<>();
        mAdapter = new AppSelectionAdapter(this, mAppList, this);
        mRecyclerView.setAdapter(mAdapter);

        getPreferences();
        updateListView();
    }

    private void getPreferences() {
        SharedPreferences prifiPrefs = getSharedPreferences(getString(R.string.prifi_config_shared_preferences), MODE_PRIVATE);
        sortField = AppListHelper.Sort.valueOf(prifiPrefs.getString(getString(R.string.prifi_ui_sort_field), String.valueOf(AppListHelper.Sort.LABEL)));
        sortDescending = prifiPrefs.getBoolean(getString(R.string.prifi_ui_sort_descending), false);
        showSystemApps = prifiPrefs.getBoolean(getString(R.string.prifi_ui_show_system_apps), false);
    }

    @Override
    public boolean onCreateOptionsMenu(Menu menu) {
        getMenuInflater().inflate(R.menu.menu_app_selection, menu);

        MenuItem showSystemAppsMenuItem = menu.findItem(R.id.show_system_apps);
        showSystemAppsMenuItem.setChecked(showSystemApps);

        MenuItem sortMenuItem = null;

        switch (sortField) {
            case LABEL:
                sortMenuItem = menu.findItem(R.id.sort_app_name);
                break;
            case PACKAGE_NAME:
                sortMenuItem = menu.findItem(R.id.sort_package_name);
                break;
        }

        int accent = getResources().getColor(R.color.colorAccent);
        Drawable arrow = getResources().getDrawable(sortDescending ?
                R.drawable.ic_arrow_downward_white_24dp :
                R.drawable.ic_arrow_upward_white_24dp);

        arrow.mutate().setColorFilter(accent, PorterDuff.Mode.SRC_IN);

        sortMenuItem.setIcon(arrow);

        searchView = (SearchView) menu.findItem(R.id.app_bar_search).getActionView();


        searchView.setOnQueryTextListener(this);

        return true;
    }

    @Override
    public boolean onOptionsItemSelected(MenuItem item) {
        int id = item.getItemId();

        switch (id) {
            case R.id.sort_app_name:
                sort(AppListHelper.Sort.LABEL);
                break;
            case R.id.sort_package_name:
                sort(AppListHelper.Sort.PACKAGE_NAME);
                break;
            case R.id.show_system_apps:
                item.setChecked(!item.isChecked());
                showSystemApps(item.isChecked());
                break;
            case R.id.on_all:
                allAppsUsePrifi(true);
                break;
            case R.id.off_all:
                allAppsUsePrifi(false);
                break;
        }

        return super.onOptionsItemSelected(item);
    }

    private void showSystemApps(boolean checked) {
        showSystemApps = checked;
        SharedPreferences prifiPrefs = getSharedPreferences(getString(R.string.prifi_config_shared_preferences), MODE_PRIVATE);
        SharedPreferences.Editor editor = prifiPrefs.edit();
        editor.putBoolean(getString(R.string.prifi_ui_show_system_apps), showSystemApps);
        editor.apply();
        updateListView();
    }

    private void sort(AppListHelper.Sort newSort) {
        SharedPreferences prifiPrefs = getSharedPreferences(getString(R.string.prifi_config_shared_preferences), MODE_PRIVATE);
        SharedPreferences.Editor editor = prifiPrefs.edit();
        //noinspection SimplifiableIfStatement
        if (sortField == newSort) {
            sortDescending = !sortDescending; // Toggle order
        } else {
            sortDescending = false;
        }

        sortField = newSort;
        editor.putString(getString(R.string.prifi_ui_sort_field), sortField.name());
        editor.putBoolean(getString(R.string.prifi_ui_sort_descending), sortDescending);
        editor.apply();

        invalidateOptionsMenu();

        updateListView();
    }

    private void allAppsUsePrifi(boolean usePrifi) {
        for (AppInfo info : mAppList) {
            info.usePrifi = usePrifi;
        }
        savePrifiApps();
        updateListView();
    }

    @Override
    public void onChecked(int position, boolean isChecked) {
        AppInfo info = mAppList.get(position);
        info.usePrifi = isChecked;
    }

    private void updateListView() {
        if (currentUpdate != null && !currentUpdate.isCancelled()) {
            currentUpdate.cancel(true);
        }
        currentUpdate = new UpdateAppListTask(this, showSystemApps, sortField, sortDescending);
        currentUpdate.execute();
    }

    protected void executeUpdateListView(List<AppInfo> appInfoList) {
        currentUpdate = null;
        mAppList.clear();
        mAppList.addAll(appInfoList);
        mAdapter.notifyDataSetChanged();
    }

    static class UpdateAppListTask extends AsyncTask<Void, Void, List<AppInfo>> {
        private final WeakReference<AppSelectionActivity> activity;
        private final boolean showSystemApps;
        private final AppListHelper.Sort sortField;
        private final boolean sortDescending;

        UpdateAppListTask(AppSelectionActivity activity, boolean showSystemApps,
                          AppListHelper.Sort sortField,
                          boolean sortDescending) {
            this.activity = new WeakReference<>(activity);
            this.showSystemApps = showSystemApps;
            this.sortField = sortField;
            this.sortDescending = sortDescending;
        }

        @Override
        protected List<AppInfo> doInBackground(Void... objects) {
            return AppListHelper.getApps(activity.get(), sortField, sortDescending, showSystemApps);
        }

        @Override
        protected void onPreExecute() {
            super.onPreExecute();
        }

        @Override
        protected void onPostExecute(List<AppInfo> appInfos) {
            activity.get().executeUpdateListView(appInfos);
        }
    }

    private void savePrifiApps() {
        List<String> prifiApps = new ArrayList<>();
        for (AppInfo info : mAppList) {
            if (info.usePrifi) {
                prifiApps.add(info.packageName);
            }
        }
        Log.i(TAG, "Saving " + prifiApps.size() + " prifi apps in preferences");
        AppListHelper.savePrifiApps(this, prifiApps);
    }

    @Override
    protected void onPause() {
        savePrifiApps();
        super.onPause();
    }

    @Override
    protected void onDestroy() {
        if (currentUpdate != null && !currentUpdate.isCancelled()) {
            currentUpdate.cancel(true);
        }
        super.onDestroy();
    }

    @Override
    public boolean onQueryTextSubmit(String query) {
        mAdapter.getFilter().filter(query);
        return false;
    }

    @Override
    public boolean onQueryTextChange(String newText) {
        mAdapter.getFilter().filter(newText);
        return false;
    }
}
