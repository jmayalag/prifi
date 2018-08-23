package ch.epfl.prifiproxy.ui;

import android.content.Context;
import android.support.design.widget.NavigationView;

/**
 * Interface to support different options in the Drawer
 */
public interface DrawerRouter {
    /**
     * Allows for overriding default Drawer Actions
     *
     * @param id      resId for the MenuItem
     * @param context context from where it is called
     * @return true if an action was performed (i.e. the option was overriden),
     * false if the action should be deferred to the caller
     */
    boolean selected(int id, Context context);

    /**
     * Allows to add additional menu items
     *
     * @param navigationView the {@link NavigationView} to which items will be added
     */
    void addMenu(NavigationView navigationView);
}
