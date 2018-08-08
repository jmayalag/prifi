package ch.epfl.prifiproxy.activities;

import android.app.ProgressDialog;
import android.content.BroadcastReceiver;
import android.content.Context;
import android.content.Intent;
import android.content.IntentFilter;
import android.content.SharedPreferences;
import android.net.VpnService;
import android.os.AsyncTask;
import android.os.Bundle;
import android.support.annotation.NonNull;
import android.support.design.widget.NavigationView;
import android.support.v4.view.GravityCompat;
import android.support.v4.widget.DrawerLayout;
import android.support.v7.app.ActionBarDrawerToggle;
import android.support.v7.app.AppCompatActivity;
import android.support.v7.widget.Toolbar;
import android.util.Log;
import android.view.MenuItem;
import android.widget.Button;
import android.widget.Toast;

import java.lang.ref.WeakReference;
import java.util.concurrent.atomic.AtomicBoolean;

import ch.epfl.prifiproxy.R;
import ch.epfl.prifiproxy.services.PrifiService;
import ch.epfl.prifiproxy.utils.HttpThroughPrifiTask;
import ch.epfl.prifiproxy.utils.NetworkHelper;
import ch.epfl.prifiproxy.utils.SystemHelper;
import eu.faircode.netguard.ServiceSinkhole;
import eu.faircode.netguard.Util;
import prifiMobile.PrifiMobile;

public class MainActivity extends AppCompatActivity implements NavigationView.OnNavigationItemSelectedListener {
    private static final String TAG = "PRIFI_MAIN";
    private static final int REQUEST_VPN = 100;

    public static final String ACTION_RULES_CHANGED = "eu.faircode.netguard.ACTION_RULES_CHANGED";
    public static final String ACTION_QUEUE_CHANGED = "eu.faircode.netguard.ACTION_QUEUE_CHANGED";
    public static final String EXTRA_REFRESH = "Refresh";
    public static final String EXTRA_SEARCH = "Search";
    public static final String EXTRA_RELATED = "Related";
    public static final String EXTRA_APPROVE = "Approve";
    public static final String EXTRA_LOGCAT = "Logcat";
    public static final String EXTRA_CONNECTED = "Connected";
    public static final String EXTRA_METERED = "Metered";
    public static final String EXTRA_SIZE = "Size";

    private Button startButton, stopButton, testPrifiButton;

    private AtomicBoolean isPrifiServiceRunning;

    private ProgressDialog mProgessDialog;
    private DrawerLayout drawer;

    private BroadcastReceiver mBroadcastReceiver;

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setContentView(R.layout.activity_main);
        Toolbar toolbar = findViewById(R.id.toolbar);
        setSupportActionBar(toolbar);

        // Buttons
        startButton = findViewById(R.id.startButton);
        stopButton = findViewById(R.id.stopButton);
        testPrifiButton = findViewById(R.id.testPrifiButton);

        // Drawer
        drawer = findViewById(R.id.drawer_layout);
        ActionBarDrawerToggle toggle = new ActionBarDrawerToggle(this, drawer, toolbar
                , R.string.navigation_drawer_open, R.string.navigation_drawer_close);

        drawer.addDrawerListener(toggle);
        toggle.syncState();

        NavigationView navigationView = findViewById(R.id.nav_view);
        navigationView.setNavigationItemSelectedListener(this);

        // Actions
        mBroadcastReceiver = new BroadcastReceiver() {
            @Override
            public void onReceive(Context context, Intent intent) {
                String action = intent.getAction();

                if (action != null) {
                    switch (action) {
                        case PrifiService.PRIFI_STOPPED_BROADCAST_ACTION: // Update UI after shutting down PriFi
                            if (mProgessDialog.isShowing()) {
                                mProgessDialog.dismiss();
                            }
                            updateUIInputCapability(false);
                            break;

                        default:
                            break;
                    }
                }

            }
        };

        startButton.setOnClickListener(view -> prepareVpn());

        stopButton.setOnClickListener(view -> stopPrifiService());

        testPrifiButton.setOnClickListener(view -> new HttpThroughPrifiTask().execute());
    }

    @Override
    public void onBackPressed() {
        if (drawer.isDrawerOpen(GravityCompat.START)) {
            drawer.closeDrawer(GravityCompat.START);
        } else {
            super.onBackPressed();
        }
    }

    @Override
    protected void onResume() {
        super.onResume();

        // Check if the PriFi service is running or not
        // Depending on the result, update UI
        isPrifiServiceRunning = new AtomicBoolean(SystemHelper.isMyServiceRunning(PrifiService.class, this));
        updateUIInputCapability(isPrifiServiceRunning.get());

        // Register BroadcastReceiver
        IntentFilter intentFilter = new IntentFilter();
        intentFilter.addAction(PrifiService.PRIFI_STOPPED_BROADCAST_ACTION);
        registerReceiver(mBroadcastReceiver, intentFilter);
    }

    @Override
    protected void onPause() {
        super.onPause();
        unregisterReceiver(mBroadcastReceiver);
    }

    private void prepareVpn() {
        Intent intent = VpnService.prepare(this);
        if (intent == null) {
            Log.i(TAG, "Vpn prepared already");
            onActivityResult(REQUEST_VPN, RESULT_OK, null);
        } else {
            startActivityForResult(intent, REQUEST_VPN);
        }
    }

    /**
     * Start PriFi "Service" (if not running)
     */
    private void startPrifiService() {
        if (!isPrifiServiceRunning.get()) {
            new StartPrifiAsyncTask(this).execute();
        }
    }

    private void startVpn() {
        ServiceSinkhole.start("UI", this);
    }

    private void stopVpn() {
        ServiceSinkhole.stop("UI", this, false);
    }

    /**
     * Stop PriFi "Core" (if running), the service will be shut down by itself.
     * <p>
     * The stopping process may take 1-2 seconds, so a ProgressDialog has been added to give users some feedback.
     */
    private void stopPrifiService() {
        if (isPrifiServiceRunning.compareAndSet(true, false)) {
            mProgessDialog = ProgressDialog.show(
                    this,
                    getString(R.string.prifi_service_stopping_dialog_title),
                    getString(R.string.prifi_service_stopping_dialog_message)
            );
            PrifiMobile.stopClient(); // StopClient will make the service to shutdown by itself
            stopVpn();
        }
    }

    @Override
    protected void onActivityResult(int requestCode, int resultCode, Intent data) {
        Log.i(TAG, "onActivityResult request=" + requestCode + " result=" + requestCode + " ok=" + (resultCode == RESULT_OK));
        Util.logExtras(data);

        if (requestCode == REQUEST_VPN) {
            // Handle VPN Approval
            if (resultCode == RESULT_OK) {
                startPrifiService();
            } else {
                Toast.makeText(this, getString(R.string.msg_vpn_cancelled), Toast.LENGTH_LONG).show();
            }
        } else {
            Log.w(TAG, "Unknown activity result request=" + requestCode);
            super.onActivityResult(requestCode, resultCode, data);
        }
    }


    /**
     * Depending on the PriFi Service status, we enable or disable some UI elements.
     *
     * @param isServiceRunning Is the PriFi Service running?
     */
    private void updateUIInputCapability(boolean isServiceRunning) {
        if (isServiceRunning) {
            startButton.setEnabled(false);
            stopButton.setEnabled(true);
        } else {
            startButton.setEnabled(true);
            stopButton.setEnabled(false);
        }
    }

    /**
     * An enum that describes the network availability.
     * <p>
     * None: Both PriFi Relay and Socks Server are not available.
     * RELAY_ONLY: Socks Server is not available.
     * SOCKS_ONLY: PriFi Relay is not available.
     * BOTH: Available
     */
    private enum NetworkStatus {
        NONE,
        RELAY_ONLY,
        SOCKS_ONLY,
        BOTH
    }

    /**
     * The Async Task that
     * <p>
     * 1. Checks network availability
     * 2. Starts PriFi Service
     * 3. Updates UI
     */
    private static class StartPrifiAsyncTask extends AsyncTask<Void, Void, NetworkStatus> {

        private final int DEFAULT_PING_TIMEOUT = 3000; // 3s

        // We need this to update UI, but it's a weak reference in order to prevent the memory leak
        private WeakReference<MainActivity> activityReference;

        StartPrifiAsyncTask(MainActivity context) {
            activityReference = new WeakReference<>(context);
        }

        /**
         * Pre Async Execution
         * <p>
         * Show a ProgressDialog, because the network check may take up to 3 seconds.
         */
        @Override
        protected void onPreExecute() {
            MainActivity activity = activityReference.get();

            if (activity != null && !activity.isFinishing()) {
                activity.mProgessDialog = ProgressDialog.show(
                        activity,
                        activity.getString(R.string.check_network_dialog_title),
                        activity.getString(R.string.check_network_dialog_message));
            }
        }

        /**
         * During Async Execution
         * <p>
         * Check the network availability
         *
         * @return relay status: none, relay only, socks only or both
         */
        @Override
        protected NetworkStatus doInBackground(Void... voids) {
            MainActivity activity = activityReference.get();
            if (activity != null && !activity.isFinishing()) {
                SharedPreferences prefs = activity.getSharedPreferences(
                        activity.getString(R.string.prifi_config_shared_preferences), MODE_PRIVATE);

                String prifiRelayAddress = prefs.getString(activity.getString(R.string.prifi_config_relay_address), "");
                int prifiRelayPort = prefs.getInt(activity.getString(R.string.prifi_config_relay_port), 0);
                int prifiRelaySocksPort = prefs.getInt(activity.getString(R.string.prifi_config_relay_socks_port), 0);

                boolean isRelayAvailable = NetworkHelper.isHostReachable(
                        prifiRelayAddress,
                        prifiRelayPort,
                        DEFAULT_PING_TIMEOUT);
                boolean isSocksAvailable = NetworkHelper.isHostReachable(
                        prifiRelayAddress,
                        prifiRelaySocksPort,
                        DEFAULT_PING_TIMEOUT);

                if (isRelayAvailable && isSocksAvailable) {
                    return NetworkStatus.BOTH;
                } else if (isRelayAvailable) {
                    return NetworkStatus.RELAY_ONLY;
                } else if (isSocksAvailable) {
                    return NetworkStatus.SOCKS_ONLY;
                } else {
                    return NetworkStatus.NONE;
                }

            } else {
                return NetworkStatus.NONE;
            }
        }

        /**
         * Post Async Execution
         * <p>
         * Start PriFi Service and update UI
         *
         * @param networkStatus relay status
         */
        @Override
        protected void onPostExecute(NetworkStatus networkStatus) {
            MainActivity activity = activityReference.get();

            if (activity != null && !activity.isFinishing()) {
                if (activity.mProgessDialog.isShowing()) {
                    activity.mProgessDialog.dismiss();
                }

                switch (networkStatus) {
                    case NONE:
                        Toast.makeText(activity, activity.getString(R.string.relay_status_none), Toast.LENGTH_LONG).show();
                        break;

                    case RELAY_ONLY:
                        Toast.makeText(activity, activity.getString(R.string.relay_status_relay_only), Toast.LENGTH_LONG).show();
                        break;

                    case SOCKS_ONLY:
                        Toast.makeText(activity, activity.getString(R.string.relay_status_socks_only), Toast.LENGTH_LONG).show();
                        break;

                    case BOTH:
                        activity.isPrifiServiceRunning.set(true);
                        activity.startService(new Intent(activity, PrifiService.class));
                        activity.updateUIInputCapability(true);
                        activity.startVpn();
                        break;

                    default:
                        break;
                }
            }
        }
    }


    @Override
    public boolean onNavigationItemSelected(@NonNull MenuItem item) {
        int id = item.getItemId();
        item.setChecked(true);
        drawer.closeDrawers();

        Intent intent = null;

        switch (id) {
            case R.id.nav_apps:
                intent = new Intent(this, AppSelectionActivity.class);
                break;
            case R.id.nav_log:
                intent = new Intent(this, OnScreenLogActivity.class);
                break;
            case R.id.nav_settings:
                intent = new Intent(this, SettingsActivity.class);
                break;
            default:
                Toast.makeText(this, "Not implemented", Toast.LENGTH_SHORT).show();
        }

        if (intent != null) {
            startActivity(intent);
        }

        return true;
    }
}
