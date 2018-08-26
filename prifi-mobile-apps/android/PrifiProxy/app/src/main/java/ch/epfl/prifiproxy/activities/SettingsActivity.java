package ch.epfl.prifiproxy.activities;

import android.content.Context;
import android.content.SharedPreferences;
import android.os.Bundle;
import android.support.design.widget.TextInputEditText;
import android.support.v7.app.AppCompatActivity;
import android.util.Log;
import android.view.KeyEvent;
import android.view.inputmethod.EditorInfo;
import android.view.inputmethod.InputMethodManager;
import android.widget.Button;
import android.widget.Switch;
import android.widget.TextView;
import android.widget.Toast;

import com.jakewharton.processphoenix.ProcessPhoenix;

import java.util.Objects;

import ch.epfl.prifiproxy.R;
import ch.epfl.prifiproxy.services.PrifiService;
import ch.epfl.prifiproxy.utils.NetworkHelper;
import ch.epfl.prifiproxy.utils.SystemHelper;
import prifiMobile.PrifiMobile;

public class SettingsActivity extends AppCompatActivity {
    private String prifiRelayAddress;
    private int prifiRelayPort;
    private int prifiRelaySocksPort;
    private boolean doDisconnectWhenNetworkError;
    private TextInputEditText relayAddressInput, relayPortInput, relaySocksPortInput;
    private Switch disconnectWhenNetworkErrorSwitch;

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setContentView(R.layout.activity_settings);
        Objects.requireNonNull(getSupportActionBar()).setDisplayHomeAsUpEnabled(true);

        // Load variables from SharedPreferences
        SharedPreferences prifiPrefs = getSharedPreferences(getString(R.string.prifi_config_shared_preferences), MODE_PRIVATE);
        prifiRelayAddress = prifiPrefs.getString(getString(R.string.prifi_config_relay_address), "");
        prifiRelayPort = prifiPrefs.getInt(getString(R.string.prifi_config_relay_port), 0);
        prifiRelaySocksPort = prifiPrefs.getInt(getString(R.string.prifi_config_relay_socks_port), 0);
        doDisconnectWhenNetworkError = prifiPrefs.getBoolean(getString(R.string.prifi_config_disconnect_when_error), false);

        Button resetButton = findViewById(R.id.resetButton);
        relayAddressInput = findViewById(R.id.relayAddressInput);
        relayPortInput = findViewById(R.id.relay_PortInput);
        relaySocksPortInput = findViewById(R.id.relaySocksPortInput);
        disconnectWhenNetworkErrorSwitch = findViewById(R.id.disconnectWhenNetworkErrorSwitch);

        resetButton.setOnClickListener(view -> resetPrifiConfig());

        relayAddressInput.setOnEditorActionListener(new DoneEditorActionListener());
        relayPortInput.setOnEditorActionListener(new DoneEditorActionListener());
        relaySocksPortInput.setOnEditorActionListener(new DoneEditorActionListener());

        disconnectWhenNetworkErrorSwitch.setOnCheckedChangeListener((buttonView, isChecked) -> applyDisconnectWhenNetworkErrorChange(isChecked));
    }

    @Override
    protected void onStart() {
        super.onStart();

        relayAddressInput.setText(prifiRelayAddress);
        relayPortInput.setText(String.valueOf(prifiRelayPort));
        relaySocksPortInput.setText(String.valueOf(prifiRelaySocksPort));
        disconnectWhenNetworkErrorSwitch.setChecked(doDisconnectWhenNetworkError);
    }

    private void applyDisconnectWhenNetworkErrorChange(boolean isChecked) {
        try {
            PrifiMobile.setMobileDisconnectWhenNetworkError(isChecked);
        } catch (Exception e) {
            throw new RuntimeException("Can't set PrifiMobile values: disconnectWhenNetworkError");
        }

        SharedPreferences.Editor editor = getSharedPreferences(getString(R.string.prifi_config_shared_preferences), MODE_PRIVATE).edit();
        editor.putBoolean(getString(R.string.prifi_config_disconnect_when_error), isChecked);
        editor.apply();

        doDisconnectWhenNetworkError = isChecked;
    }

    /**
     * Reset PriFi Configuration to its default value.
     * <p>
     * It sets Preferences.isFirstInit to true and restart the app. The Application class will do the rest.
     */
    private void resetPrifiConfig() {
        if (!SystemHelper.isMyServiceRunning(PrifiService.class, this)) {
            SharedPreferences.Editor editor = getSharedPreferences(getString(R.string.prifi_config_shared_preferences), MODE_PRIVATE).edit();
            editor.putBoolean(getString(R.string.prifi_config_first_init), true);
            editor.apply();

            ProcessPhoenix.triggerRebirth(this);
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

            case R.id.relay_PortInput:
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
