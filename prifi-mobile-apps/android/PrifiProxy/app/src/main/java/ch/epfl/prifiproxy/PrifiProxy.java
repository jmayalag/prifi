package ch.epfl.prifiproxy;

import android.app.Application;
import android.content.Context;
import android.content.SharedPreferences;

import prifiMobile.PrifiMobile;

/**
 * The entry point of this app
 * This class allows us to do some initialization work after launching the app.
 */
public class PrifiProxy extends Application {

    private static Application mApplication;

    @Override
    public void onCreate() {
        super.onCreate();
        mApplication = this;

        initPrifiConfig();
    }

    public static Context getContext() {
        return mApplication.getApplicationContext();
    }

    /**
     * PriFi Initialization
     * Retrieve, save and modify (if necessary) the important PriFi values
     */
    private void initPrifiConfig() {
        final String defaultRelayAddress;
        final int defaultRelayPort;
        final int defaultRelaySocksPort;
        final boolean defaultDoDisconnectWhenNetworkError;

        // Retrieve relay info
        try {
            defaultRelayAddress = PrifiMobile.getRelayAddress();
            defaultRelayPort = longToInt(PrifiMobile.getRelayPort());
            defaultRelaySocksPort = longToInt(PrifiMobile.getRelaySocksPort());
            defaultDoDisconnectWhenNetworkError = PrifiMobile.getMobileDisconnectWhenNetworkError();
        } catch (Exception e) {
            throw new RuntimeException("Can't read configuration files");
        }

        SharedPreferences prifiPrefs = getSharedPreferences(getString(R.string.prifi_config_shared_preferences), MODE_PRIVATE);
        Boolean isFirstInit = prifiPrefs.getBoolean(getString(R.string.prifi_config_first_init), true);

        // Save if it's the first initialization
        if (isFirstInit) {
            final String pubKey;
            final String priKey;

            try {
                PrifiMobile.generateNewKeyPairAndAssign();
                pubKey = PrifiMobile.getPublicKey();
                priKey = PrifiMobile.getPrivateKey();
            } catch (Exception e) {
                throw new RuntimeException("Can't generate keys");
            }

            SharedPreferences.Editor editor = getSharedPreferences(getString(R.string.prifi_config_shared_preferences), MODE_PRIVATE).edit();
            // Save default values
            editor.putString(getString(R.string.prifi_config_relay_address_default), defaultRelayAddress);
            editor.putInt(getString(R.string.prifi_config_relay_port_default), defaultRelayPort);
            editor.putInt(getString(R.string.prifi_config_relay_socks_port_default), defaultRelaySocksPort);
            editor.putBoolean(getString(R.string.prifi_config_disconnect_when_error_default), defaultDoDisconnectWhenNetworkError);
            // Copy default values
            editor.putString(getString(R.string.prifi_config_relay_address), defaultRelayAddress);
            editor.putInt(getString(R.string.prifi_config_relay_port), defaultRelayPort);
            editor.putInt(getString(R.string.prifi_config_relay_socks_port), defaultRelaySocksPort);
            editor.putBoolean(getString(R.string.prifi_config_disconnect_when_error), defaultDoDisconnectWhenNetworkError);
            // Save keys
            editor.putString(getString(R.string.prifi_client_public_key), pubKey);
            editor.putString(getString(R.string.prifi_client_private_key), priKey);
            // Set isFirstInit false
            editor.putBoolean(getString(R.string.prifi_config_first_init), false);
            // apply
            editor.apply();
        } else {
            // Set if not
            final String currentPrifiRelayAddress = prifiPrefs.getString(getString(R.string.prifi_config_relay_address),"");
            final int currentPrifiRelayPort = prifiPrefs.getInt(getString(R.string.prifi_config_relay_port), 0);
            final int currentPrifiRelaySocksPort = prifiPrefs.getInt(getString(R.string.prifi_config_relay_socks_port),0);

            final String currentPubKey = prifiPrefs.getString(getString(R.string.prifi_client_public_key),"");
            final String currentPriKey = prifiPrefs.getString(getString(R.string.prifi_client_private_key),"");

            final boolean currentDoDisconnectWhenNetworkError = prifiPrefs.getBoolean(getString(R.string.prifi_config_disconnect_when_error), false);

            try {

                if (!currentPrifiRelayAddress.equals(defaultRelayAddress)) {
                    PrifiMobile.setRelayAddress(currentPrifiRelayAddress);
                }

                if (currentPrifiRelayPort != defaultRelayPort) {
                    PrifiMobile.setRelayPort((long) currentPrifiRelayPort);
                }

                if (currentPrifiRelaySocksPort != defaultRelaySocksPort) {
                    PrifiMobile.setRelaySocksPort((long) currentPrifiRelaySocksPort);
                }

                if (currentDoDisconnectWhenNetworkError != defaultDoDisconnectWhenNetworkError) {
                    PrifiMobile.setMobileDisconnectWhenNetworkError(currentDoDisconnectWhenNetworkError);
                }

                PrifiMobile.setPublicKey(currentPubKey);
                PrifiMobile.setPrivateKey(currentPriKey);

            } catch (Exception e) {
                throw new RuntimeException("Can't set PrifiMobile values");
            }
        }
    }

    private int longToInt(long l) {
        if (l < Integer.MIN_VALUE || l > Integer.MAX_VALUE) {
            throw new IllegalArgumentException(l + " Invalid Port");
        }
        return (int) l;
    }

}
