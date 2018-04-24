package ch.epfl.prifiproxy.utils;

import android.content.Context;
import android.os.AsyncTask;
import android.widget.Toast;

import java.io.IOException;
import java.net.InetSocketAddress;
import java.net.Proxy;
import java.util.concurrent.TimeUnit;

import ch.epfl.prifiproxy.PrifiProxy;
import ch.epfl.prifiproxy.R;
import okhttp3.OkHttpClient;
import okhttp3.Request;
import okhttp3.Response;

/**
 * This class is an AsyncTask that makes a HTTP Request through PriFi.
 * The purpose is to test the PriFi connexion.
 * (HTTP Request + SOCKS Proxy + PriFi Localhost)
 */
public class HttpThroughPrifiTask extends AsyncTask<Void, Void, Boolean> {

    /**
     * Request the google page through PriFi (localhost:8080)
     * @return is the HTTP request successful?
     */
    @Override
    protected Boolean doInBackground(Void... voids) {
        Proxy proxy = new Proxy(Proxy.Type.SOCKS, new InetSocketAddress("127.0.0.1", 8080)); // TODO Don't hard code the port
        OkHttpClient client = new OkHttpClient.Builder().connectTimeout(5, TimeUnit.SECONDS).proxy(proxy).build();
        Request request = new Request.Builder().url("https://www.google.com").get().build();

        boolean isSuccessful;
        try {
            Response response = client.newCall(request).execute();
            isSuccessful = response.isSuccessful();
        } catch (IOException e) {
            isSuccessful = false;
        }

        return isSuccessful;
    }

    /**
     * Display the result to users.
     * @param isSuccessful is the HTTP request successful?
     */
    @Override
    protected void onPostExecute(Boolean isSuccessful) {
        Context context = PrifiProxy.getContext();
        if (isSuccessful) {
            Toast.makeText(context, context.getString(R.string.prifi_test_message_successful), Toast.LENGTH_SHORT).show();
        } else {
            Toast.makeText(context, context.getString(R.string.prifi_test_message_failed), Toast.LENGTH_SHORT).show();
        }
    }
}
