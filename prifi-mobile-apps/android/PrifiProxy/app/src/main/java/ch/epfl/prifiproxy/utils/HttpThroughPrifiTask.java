package ch.epfl.prifiproxy.utils;

import android.content.Context;
import android.os.AsyncTask;
import android.widget.Toast;

import ch.epfl.prifiproxy.PrifiProxy;
import ch.epfl.prifiproxy.R;
import prifiMobile.HttpRequestResult;

/**
 * This class is an AsyncTask that makes a HTTP Request through PriFi.
 * The purpose is to test the PriFi connexion.
 */
public class HttpThroughPrifiTask extends AsyncTask<Void, Void, Boolean> {

    /**
     * Request the google page through PriFi
     * @return is the HTTP request successful? (true if no error)
     */
    @Override
    protected Boolean doInBackground(Void... voids) {
        final String targetUrl = "https://www.google.com";
        final long timeout = 5;
        boolean isSuccessful = true;

        HttpRequestResult result = new HttpRequestResult();
        try {
            // Check golang part for more info about this method. The third parameter is whether use prifi or not
            result.retrieveHttpResponseThroughPrifi(targetUrl, timeout, true);
        } catch (Exception e) {
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
