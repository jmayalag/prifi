package ch.epfl.prifiproxy.listeners;

public interface OnItemCheckedListener<T> {
    void onChecked(T item, boolean isChecked);
}
