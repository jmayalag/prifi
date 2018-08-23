package ch.epfl.prifiproxy.ui;

import android.content.Context;
import android.content.Intent;
import android.support.design.widget.NavigationView;
import android.widget.Toast;

import ch.epfl.prifiproxy.R;
import ch.epfl.prifiproxy.activities.AppSelectionActivity;
import ch.epfl.prifiproxy.activities.SettingsActivity;

public class MainDrawerRouter implements DrawerRouter {
    @Override
    public boolean selected(int id, Context context) {
        return false;
    }

    public void addMenu(NavigationView navigationView) {

    }
}
