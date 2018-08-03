package ch.epfl.prifiproxy.activities;

import android.os.Bundle;
import android.support.v7.app.AppCompatActivity;
import android.support.v7.widget.LinearLayoutManager;
import android.support.v7.widget.RecyclerView;
import android.support.v7.widget.Toolbar;
import android.util.Log;
import android.view.Menu;
import android.view.MenuItem;

import java.util.ArrayList;
import java.util.List;

import ch.epfl.prifiproxy.R;
import ch.epfl.prifiproxy.adapters.AppSelectionAdapter;
import ch.epfl.prifiproxy.listeners.OnAppCheckedListener;
import ch.epfl.prifiproxy.utils.AppInfo;
import ch.epfl.prifiproxy.utils.AppListHelper;

public class AppSelectionActivity extends AppCompatActivity implements OnAppCheckedListener {
    private static final String TAG = "PRIFI_APP_SELECTION";
    private RecyclerView mRecyclerView;
    private RecyclerView.Adapter mAdapter;
    private RecyclerView.LayoutManager mLayoutManager;
    private List<AppInfo> mAppList;


    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setContentView(R.layout.activity_app_selection);
        Toolbar toolbar = findViewById(R.id.toolbar);
        setSupportActionBar(toolbar);

        getSupportActionBar().setDisplayHomeAsUpEnabled(true);

        mRecyclerView = findViewById(R.id.app_list);
        mRecyclerView.setHasFixedSize(true);

        mLayoutManager = new LinearLayoutManager(this);
        mRecyclerView.setLayoutManager(mLayoutManager);

        mAppList = AppListHelper.getApps(this);
        mAdapter = new AppSelectionAdapter(this, mAppList, this);
        mRecyclerView.setAdapter(mAdapter);
    }

    @Override
    public boolean onCreateOptionsMenu(Menu menu) {
        getMenuInflater().inflate(R.menu.menu_app_selection, menu);
        return true;
    }

    @Override
    public boolean onOptionsItemSelected(MenuItem item) {
        int id = item.getItemId();

        switch (id) {
            case R.id.on_all:
                allAppsUsePrifi(true);
                break;
            case R.id.off_all:
                allAppsUsePrifi(false);
                break;
        }

        return super.onOptionsItemSelected(item);
    }

    private void allAppsUsePrifi(boolean usePrifi) {
        for (AppInfo info : mAppList) {
            info.usePrifi = usePrifi;
        }
        updateListView();
    }

    @Override
    public void onChecked(int position, boolean isChecked) {
        AppInfo info = mAppList.get(position);
        Log.d(TAG, info.packageName + " isChecked: " + isChecked);
        info.usePrifi = isChecked;
    }

    private void updateListView() {
        mAdapter.notifyDataSetChanged();
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
}
