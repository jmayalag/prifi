package com.example.junchen.prifiproxy.activities;

import android.app.ActivityManager;
import android.content.Context;
import android.content.Intent;
import android.os.Bundle;
import android.support.v7.app.AppCompatActivity;
import android.util.Log;
import android.view.View;
import android.widget.Button;

import com.example.junchen.prifiproxy.R;
import com.example.junchen.prifiproxy.services.PrifiService;

import java.util.concurrent.atomic.AtomicBoolean;

import prifiMobile.PrifiMobile;

public class MainActivity extends AppCompatActivity {

    private AtomicBoolean isPrifiServiceRunning;
    private Button startButton;
    private Button stopButton;
    private Button testButton;

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setContentView(R.layout.activity_main);

        isPrifiServiceRunning = new AtomicBoolean(isMyServiceRunning(PrifiService.class));
        startButton = findViewById(R.id.startButton);
        stopButton = findViewById(R.id.stopButton);
        testButton = findViewById(R.id.testButton);

        startButton.setOnClickListener(new View.OnClickListener() {
            @Override
            public void onClick(View view) {
                startPrifiService();
            }
        });

        stopButton.setOnClickListener(new View.OnClickListener() {
            @Override
            public void onClick(View view) {
                stopPrifiService();
            }
        });

        testButton.setOnClickListener(new View.OnClickListener() {
            @Override
            public void onClick(View view) {
                boolean b = isMyServiceRunning(PrifiService.class);
                Log.i("myapp", String.valueOf(b));
            }
        });
    }

    private void startPrifiService() {
        if (isPrifiServiceRunning.compareAndSet(false, true)) {
            startService(new Intent(this, PrifiService.class));
        }
    }

    private void stopPrifiService() {
        if (isPrifiServiceRunning.compareAndSet(true, false)) {
            PrifiMobile.stopClient(); // StopClient will make the service to shutdown by itself
        }
    }

    private boolean isMyServiceRunning(Class<?> serviceClass) {
        ActivityManager manager = (ActivityManager) getSystemService(Context.ACTIVITY_SERVICE);
        for (ActivityManager.RunningServiceInfo service : manager.getRunningServices(Integer.MAX_VALUE)) {
            if (serviceClass.getName().equals(service.service.getClassName())) {
                return true;
            }
        }
        return false;
    }

}
