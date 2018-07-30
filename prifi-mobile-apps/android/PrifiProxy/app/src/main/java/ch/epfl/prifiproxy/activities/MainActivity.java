package ch.epfl.prifiproxy.activities;

import android.app.ActivityOptions;
import android.app.AlertDialog;
import android.app.ProgressDialog;
import android.content.BroadcastReceiver;
import android.content.Context;
import android.content.Intent;
import android.content.IntentFilter;
import android.content.SharedPreferences;
import android.net.Uri;
import android.os.AsyncTask;
import android.os.Build;
import android.os.Bundle;
import android.support.design.widget.TextInputEditText;
import android.support.v7.app.AppCompatActivity;
import android.support.v7.widget.Toolbar;
import android.view.KeyEvent;
import android.view.Menu;
import android.view.MenuItem;
import android.view.inputmethod.EditorInfo;
import android.view.inputmethod.InputMethodManager;
import android.widget.Button;
import android.widget.TextView;
import android.widget.Toast;

import com.jakewharton.processphoenix.ProcessPhoenix;

import java.lang.ref.WeakReference;
import java.util.concurrent.atomic.AtomicBoolean;

import ch.epfl.prifiproxy.R;
import ch.epfl.prifiproxy.services.PrifiService;
import ch.epfl.prifiproxy.utils.HttpThroughPrifiTask;
import ch.epfl.prifiproxy.utils.NetworkHelper;
import ch.epfl.prifiproxy.utils.SystemHelper;
import prifiMobile.PrifiMobile;

public class MainActivity extends AppCompatActivity {

    private String prifiRelayAddress;
    private int prifiRelayPort;
    private int prifiRelaySocksPort;

    private AtomicBoolean isPrifiServiceRunning;

    private Button startButton, stopButton, resetButton, testPrifiButton, logButton;
    private TextInputEditText relayAddressInput, relayPortInput, relaySocksPortInput;
    private ProgressDialog mProgessDialog;

    private BroadcastReceiver mBroadcastReceiver;

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setContentView(R.layout.activity_main);
        Toolbar toolbar = findViewById(R.id.toolbar);
        setSupportActionBar(toolbar);

        // Load Variables from SharedPreferences
        SharedPreferences prifiPrefs = getSharedPreferences(getString(R.string.prifi_config_shared_preferences), MODE_PRIVATE);
        prifiRelayAddress = prifiPrefs.getString(getString(R.string.prifi_config_relay_address), "");
        prifiRelayPort = prifiPrefs.getInt(getString(R.string.prifi_config_relay_port), 0);
        prifiRelaySocksPort = prifiPrefs.getInt(getString(R.string.prifi_config_relay_socks_port), 0);

        // Buttons
        startButton = findViewById(R.id.startButton);
        stopButton = findViewById(R.id.stopButton);
        resetButton = findViewById(R.id.resetButton);
        testPrifiButton = findViewById(R.id.testPrifiButton);
        logButton = findViewById(R.id.logButton);
        relayAddressInput = findViewById(R.id.relayAddressInput);
        relayPortInput = findViewById(R.id.relayPortInput);
        relaySocksPortInput = findViewById(R.id.relaySocksPortInput);

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

        startButton.setOnClickListener(view -> startPrifiService());

        stopButton.setOnClickListener(view -> stopPrifiService());

        resetButton.setOnClickListener(view -> resetPrifiConfig());

        relayAddressInput.setText(prifiRelayAddress);
        relayAddressInput.setOnEditorActionListener(new DoneEditorActionListener());

        relayPortInput.setText(String.valueOf(prifiRelayPort));
        relayPortInput.setOnEditorActionListener(new DoneEditorActionListener());

        relaySocksPortInput.setText(String.valueOf(prifiRelaySocksPort));
        relaySocksPortInput.setOnEditorActionListener(new DoneEditorActionListener());

        testPrifiButton.setOnClickListener(view -> new HttpThroughPrifiTask().execute());

        logButton.setOnClickListener(view -> {
            Intent intent = new Intent(this, OnScreenLogActivity.class);
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.LOLLIPOP) {
                startActivity(intent, ActivityOptions.makeSceneTransitionAnimation(this).toBundle());
            } else {
                startActivity(intent);
            }
        });
    }

    @Override
    public boolean onCreateOptionsMenu(Menu menu) {
        getMenuInflater().inflate(R.menu.menu_main, menu);
        return true;
    }

    @Override
    public boolean onOptionsItemSelected(MenuItem item) {
        int id = item.getItemId();

        //noinspection SimplifiableIfStatement
        if (id == R.id.app_selection) {
            startActivity(new Intent(this, AppSelectionActivity.class));
            return true;
        }

        return super.onOptionsItemSelected(item);
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

    /**
     * Start PriFi "Service" (if not running)
     * <p>
     * It will execute an AsyncTask, because the network check can't be on the main thread.
     */
    private void startPrifiService() {
        if (!isPrifiServiceRunning.get()) {
            new StartPrifiAsyncTask(this).execute();
        }
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
        }
    }

    /**
     * A Dialog that guides users to launch or install Telegram after enabling PriFi Service
     */
    private void showRedirectDialog() {
        AlertDialog alertDialog = new AlertDialog.Builder(this).create();
        alertDialog.setTitle(getString(R.string.redirect_dialog_title));
        alertDialog.setMessage(getString(R.string.redirect_dialog_message));
        alertDialog.setButton(AlertDialog.BUTTON_NEGATIVE, getString(R.string.redirect_dialog_cancel),
                (dialog, which) -> dialog.dismiss());
        alertDialog.setButton(AlertDialog.BUTTON_POSITIVE, getString(R.string.redirect_dialog_confirm),
                (dialog, which) -> redirectToTelegram());
        alertDialog.show();
    }

    /**
     * Open Telegram if the app is installed, otherwise open Google Play Download Page.
     */
    private void redirectToTelegram() {
        final String appName = "org.telegram.messenger";
        Intent intent;
        final boolean isAppInstalled = SystemHelper.isAppAvailable(this, appName);
        if (isAppInstalled) {
            intent = getPackageManager().getLaunchIntentForPackage(appName);
        } else {
            intent = new Intent(Intent.ACTION_VIEW);
            intent.setData(Uri.parse("market://details?id=" + appName));
        }
        startActivity(intent);
    }

    /**
     * Trigger actions if the Done key is pressed
     *
     * @param view the input field where the Done key is pressed
     */
    private void triggerDoneAction(TextView view) {
        String text = view.getText().toString();
        switch (view.getId()) {
            case R.id.relayAddressInput:
                updateInputFieldsAndPrefs(text, null, null);
                break;

            case R.id.relayPortInput:
                updateInputFieldsAndPrefs(null, text, null);
                break;

            case R.id.relaySocksPortInput:
                updateInputFieldsAndPrefs(null, null, text);
                break;

            default:
                break;
        }
    }

    /**
     * Update input fields and preferences, if the user input is valid.
     *
     * @param relayAddressText   user input relay address
     * @param relayPortText      user input relay port
     * @param relaySocksPortText user input relay socks port
     */
    private void updateInputFieldsAndPrefs(String relayAddressText, String relayPortText, String relaySocksPortText) {
        SharedPreferences.Editor editor = getSharedPreferences(getString(R.string.prifi_config_shared_preferences), MODE_PRIVATE).edit();

        try {

            if (relayAddressText != null) {
                if (NetworkHelper.isValidIpv4Address(relayAddressText)) {
                    prifiRelayAddress = relayAddressText;
                    editor.putString(getString(R.string.prifi_config_relay_address), prifiRelayAddress);

                    PrifiMobile.setRelayAddress(prifiRelayAddress);
                } else {
                    Toast.makeText(this, getString(R.string.prifi_invalid_address), Toast.LENGTH_SHORT).show();
                }
                relayAddressInput.setText(prifiRelayAddress);
            }

            if (relayPortText != null) {
                if (NetworkHelper.isValidPort(relayPortText)) {
                    prifiRelayPort = Integer.parseInt(relayPortText);
                    editor.putInt(getString(R.string.prifi_config_relay_port), prifiRelayPort);

                    PrifiMobile.setRelayPort((long) prifiRelayPort);
                } else {
                    Toast.makeText(this, getString(R.string.prifi_invalid_port), Toast.LENGTH_SHORT).show();
                }
                relayPortInput.setText(String.valueOf(prifiRelayPort));
            }

            if (relaySocksPortText != null) {
                if (NetworkHelper.isValidPort(relaySocksPortText)) {
                    prifiRelaySocksPort = Integer.parseInt(relaySocksPortText);
                    editor.putInt(getString(R.string.prifi_config_relay_socks_port), prifiRelaySocksPort);

                    PrifiMobile.setRelaySocksPort((long) prifiRelaySocksPort);
                } else {
                    Toast.makeText(this, getString(R.string.prifi_invalid_port), Toast.LENGTH_SHORT).show();
                }
                relaySocksPortInput.setText(String.valueOf(prifiRelaySocksPort));
            }

        } catch (Exception e) {
            e.printStackTrace();
            Toast.makeText(this, getString(R.string.prifi_configuration_failed), Toast.LENGTH_LONG).show();
        } finally {
            editor.apply();
        }

    }

    /**
     * Reset PriFi Configuration to its default value.
     * <p>
     * It sets Preferences.isFirstInit to true and restart the app. The Application class will do the rest.
     */
    private void resetPrifiConfig() {
        if (!isPrifiServiceRunning.get()) {
            SharedPreferences.Editor editor = getSharedPreferences(getString(R.string.prifi_config_shared_preferences), MODE_PRIVATE).edit();
            editor.putBoolean(getString(R.string.prifi_config_first_init), true);
            editor.apply();

            ProcessPhoenix.triggerRebirth(this);
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
            resetButton.setEnabled(false);
            relayAddressInput.setEnabled(false);
            relayPortInput.setEnabled(false);
            relaySocksPortInput.setEnabled(false);
        } else {
            startButton.setEnabled(true);
            stopButton.setEnabled(false);
            resetButton.setEnabled(true);
            relayAddressInput.setEnabled(true);
            relayPortInput.setEnabled(true);
            relaySocksPortInput.setEnabled(true);
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
                boolean isRelayAvailable = NetworkHelper.isHostReachable(
                        activity.prifiRelayAddress,
                        activity.prifiRelayPort,
                        DEFAULT_PING_TIMEOUT);
                boolean isSocksAvailable = NetworkHelper.isHostReachable(
                        activity.prifiRelayAddress,
                        activity.prifiRelaySocksPort,
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
                        activity.showRedirectDialog();
                        break;

                    default:
                        break;
                }
            }
        }
    }

    /**
     * A custom EditorActionListener
     * <p>
     * When the Done key is pressed, execute pre defined actions and hide the virtual keyboard.
     */
    private class DoneEditorActionListener implements TextView.OnEditorActionListener {
        @Override
        public boolean onEditorAction(TextView textView, int actionId, KeyEvent keyEvent) {
            if (actionId == EditorInfo.IME_ACTION_DONE) {
                triggerDoneAction(textView);
                InputMethodManager imm = (InputMethodManager) textView.getContext().getSystemService(Context.INPUT_METHOD_SERVICE);
                if (imm != null) {
                    imm.hideSoftInputFromWindow(textView.getWindowToken(), 0);
                }
                return true;
            }
            return false;
        }
    }

}
