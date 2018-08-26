package ch.epfl.prifiproxy.listeners;

public interface OnItemGestureListener<T> {
    void itemMoved(T item, T afterItem, int fromPosition, int toPosition);

    void itemSwiped(T item);
}
