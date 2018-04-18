package ch.epfl.prifiproxy.activities;

import android.app.ActivityManager;
import android.app.AlertDialog;
import android.app.ProgressDialog;
import android.content.BroadcastReceiver;
import android.content.Context;
import android.content.Intent;
import android.content.IntentFilter;
import android.content.SharedPreferences;
import android.content.pm.PackageManager;
import android.net.Uri;
import android.os.AsyncTask;
import android.os.Bundle;
import android.support.design.widget.TextInputEditText;
import android.support.v7.app.AppCompatActivity;
import android.view.KeyEvent;
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
import ch.epfl.prifiproxy.utils.NetworkHelper;
import ch.epfl.prifiproxy.utils.SystemHelper;
import prifiMobile.PrifiMobile;

public class MainActivity extends AppCompatActivity {

    private String prifiRelayAddress;
    private int prifiRelayPort;
    private int prifiRelaySocksPort;

    private AtomicBoolean isPrifiServiceRunning;

    private Button startButton, stopButton, resetButton, testButton1, testButton2;
    private TextInputEditText relayAddressInput, relayPortInput, relaySocksPortInput;
    private ProgressDialog mProgessDialog;

    private BroadcastReceiver mBroadcastReceiver;

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setContentView(R.layout.activity_main);

        // Load Variables from SharedPreferences
        SharedPreferences prifiPrefs = getSharedPreferences(getString(R.string.prifi_config_shared_preferences), MODE_PRIVATE);
        prifiRelayAddress = prifiPrefs.getString(getString(R.string.prifi_config_relay_address),"");
        prifiRelayPort = prifiPrefs.getInt(getString(R.string.prifi_config_relay_port), 0);
        prifiRelaySocksPort = prifiPrefs.getInt(getString(R.string.prifi_config_relay_socks_port),0);

        // Buttons
        startButton = findViewById(R.id.startButton);
        stopButton = findViewById(R.id.stopButton);
        resetButton = findViewById(R.id.resetButton);
        testButton1 = findViewById(R.id.testButton1);
        testButton2 = findViewById(R.id.testButton2);
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

        testButton1.setOnClickListener(view -> {

        });

        testButton2.setOnClickListener(view -> {

        });
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
     *
     * It will execute an AsyncTask, because the network check can't be on the main thread.
     */
    private void startPrifiService() {
        if (!isPrifiServiceRunning.get()) {
            new StartPrifiAsyncTask(this).execute();
        }
    }

    /**
     * Stop PriFi "Core" (if running), the service will be shut down by itself.
     *
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
     * @param relayAddressText user input relay address
     * @param relayPortText user input relay port
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
     *
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
     * The Async Task that
     *
     * 1. Checks network availability
     * 2. Starts PriFi Service
     * 3. Updates UI
     */
    private static class StartPrifiAsyncTask extends AsyncTask<Void, Void, Boolean> {

        private final int DEFAULT_PING_TIMEOUT = 3000; // 3s

        // We need this to update UI, but it's a weak reference in order to prevent the memory leak
        private WeakReference<MainActivity> activityReference;

        StartPrifiAsyncTask(MainActivity context) {
            activityReference = new WeakReference<>(context);
        }

        /**
         * Pre Async Execution
         *
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
         *
         * Check the network availability
         *
         * @return true if the relay is available, otherwise false
         */
        @Override
        protected Boolean doInBackground(Void... voids) {
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
                return isRelayAvailable && isSocksAvailable;
            } else {
                return false;
            }
        }

        /**
         * Post Async Execution
         *
         * Start PriFi Service and update UI
         *
         * @param isNetworkAvailable Is the relay available?
         */
        @Override
        protected void onPostExecute(Boolean isNetworkAvailable) {
            MainActivity activity = activityReference.get();

            if (activity != null && !activity.isFinishing()) {
                if (activity.mProgessDialog.isShowing()) {
                    activity.mProgessDialog.dismiss();
                }

                if (isNetworkAvailable) {
                    activity.isPrifiServiceRunning.set(true);
                    activity.startService(new Intent(activity, PrifiService.class));
                    activity.updateUIInputCapability(true);
                    activity.showRedirectDialog();
                } else {
                    Toast.makeText(activity, activity.getString(R.string.relay_not_available), Toast.LENGTH_LONG).show();
                }
            }
        }
    }

    /**
     * A custom EditorActionListener
     *
     * When the Done key is pressed, execute pre defined actions and hide the virtual keyboard.
     */
    private class DoneEditorActionListener implements TextView.OnEditorActionListener {
        @Override
        public boolean onEditorAction(TextView textView, int actionId, KeyEvent keyEvent) {
            if (actionId == EditorInfo.IME_ACTION_DONE) {
                triggerDoneAction(textView);
                InputMethodManager imm = (InputMethodManager)textView.getContext().getSystemService(Context.INPUT_METHOD_SERVICE);
                if (imm != null) {
                    imm.hideSoftInputFromWindow(textView.getWindowToken(), 0);
                }
                return true;
            }
            return false;
        }
    }

}
