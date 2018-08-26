package ch.epfl.prifiproxy.ui;

import android.support.annotation.IntDef;

import java.lang.annotation.Retention;

import static java.lang.annotation.RetentionPolicy.SOURCE;

/**
 * Set of UI modes
 */
@Retention(SOURCE)
@IntDef({Mode.ADD, Mode.EDIT, Mode.DETAIL})
public @interface Mode {
    int ADD = 0;
    int EDIT = 1;
    int DETAIL = 2;
}