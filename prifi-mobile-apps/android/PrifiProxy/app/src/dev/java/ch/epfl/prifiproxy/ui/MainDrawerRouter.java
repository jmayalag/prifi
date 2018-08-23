package ch.epfl.prifiproxy.ui;

import android.content.Context;
import android.content.Intent;
import android.support.design.widget.NavigationView;

import ch.epfl.prifiproxy.R;
import ch.epfl.prifiproxy.activities.OnScreenLogActivity;

public class MainDrawerRouter implements DrawerRouter {
    @Override
    public boolean selected(int id, Context context) {
        Intent intent = null;

        switch (id) {
            case R.id.nav_log:
                intent = new Intent(context, OnScreenLogActivity.class);
                break;
        }

        if (intent != null) {
            context.startActivity(intent);
            return true;
        }
        return false;
    }

    public void addMenu(NavigationView navigationView) {
        navigationView.inflateMenu(R.menu.activity_main_drawer_extra);
    }
}
