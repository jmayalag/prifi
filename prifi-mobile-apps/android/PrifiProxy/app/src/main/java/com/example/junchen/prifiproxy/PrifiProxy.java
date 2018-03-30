package com.example.junchen.prifiproxy;

import android.app.Application;
import android.content.Context;

/**
 * Created by junchen on 23.03.18.
 */

public class PrifiProxy extends Application {

    private static Context mContext;

    @Override
    public void onCreate() {
        super.onCreate();
        mContext = getApplicationContext();
    }

    public static Context getContext() {
        return mContext;
    }

}
