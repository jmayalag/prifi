package ch.epfl.prifiproxy.adapters;

import android.annotation.SuppressLint;
import android.graphics.Color;
import android.support.annotation.ColorInt;
import android.support.annotation.NonNull;
import android.support.annotation.Nullable;
import android.support.v4.view.MotionEventCompat;
import android.support.v7.widget.RecyclerView;
import android.support.v7.widget.helper.ItemTouchHelper;
import android.view.LayoutInflater;
import android.view.MotionEvent;
import android.view.View;
import android.view.ViewGroup;
import android.widget.ImageView;
import android.widget.TextView;

import java.util.ArrayList;
import java.util.Collections;
import java.util.List;

import ch.epfl.prifiproxy.R;
import ch.epfl.prifiproxy.listeners.OnItemClickListener;
import ch.epfl.prifiproxy.listeners.OnItemGestureListener;
import ch.epfl.prifiproxy.persistence.entity.Configuration;
import ch.epfl.prifiproxy.ui.SwipeAndDragHelper;
import ch.epfl.prifiproxy.ui.helper.ItemTouchHelperAdapter;
import ch.epfl.prifiproxy.ui.helper.ItemTouchHelperViewHolder;
import ch.epfl.prifiproxy.ui.helper.OnStartDragListener;

public class ConfigurationRecyclerAdapter extends RecyclerView.Adapter<ConfigurationRecyclerAdapter.ViewHolder>
        implements ItemTouchHelperAdapter {
    @Nullable
    private final OnItemClickListener<Configuration> clickListener;
    @Nullable
    private final OnItemGestureListener<Configuration> gestureListener;
    @Nullable
    private final OnStartDragListener startDragListener;
    private final boolean isEditMode;

    @NonNull
    private List<Configuration> dataset;

    public ConfigurationRecyclerAdapter(@Nullable OnItemClickListener<Configuration> clickListener,
                                        @Nullable OnItemGestureListener<Configuration> gestureListener,
                                        @Nullable OnStartDragListener startDragListener,
                                        boolean isEditMode) {
        this.dataset = new ArrayList<>();
        this.clickListener = clickListener;
        this.gestureListener = gestureListener;
        this.startDragListener = startDragListener;
        this.isEditMode = isEditMode;
    }

    @NonNull
    @Override
    public ViewHolder onCreateViewHolder(@NonNull ViewGroup parent, int viewType) {
        View v = LayoutInflater.from(parent.getContext())
                .inflate(R.layout.configuration_list_item, parent, false);

        return new ViewHolder(v, clickListener, gestureListener, startDragListener, isEditMode);
    }

    @SuppressLint("ClickableViewAccessibility")
    @Override
    public void onBindViewHolder(@NonNull ViewHolder holder, int position) {
        Configuration item = dataset.get(position);
        holder.bind(item);
    }

    @Override
    public int getItemCount() {
        return dataset.size();
    }

    public void setData(@NonNull List<Configuration> dataset) {
        this.dataset = dataset;
        notifyDataSetChanged();
    }

    @Override
    public void onItemDismiss(int position) {
        if (gestureListener != null) {
            gestureListener.itemSwiped(dataset.get(position));
        }
        dataset.remove(position);
        notifyItemRemoved(position);
    }

    @Override
    public boolean onItemMove(int fromPosition, int toPosition) {
        Collections.swap(dataset, fromPosition, toPosition);
        notifyItemMoved(fromPosition, toPosition);
        return true;
    }

    @NonNull
    public List<Configuration> getDataset() {
        return dataset;
    }

    static class ViewHolder extends RecyclerView.ViewHolder implements ItemTouchHelperViewHolder {
        TextView configurationName;
        TextView configurationHost;
        ImageView reorderView;
        Configuration item;

        @SuppressLint("ClickableViewAccessibility")
        ViewHolder(View itemView,
                   @Nullable OnItemClickListener<Configuration> clickListener,
                   @Nullable OnItemGestureListener<Configuration> gestureListener,
                   @Nullable OnStartDragListener dragListener,
                   boolean isEditMode) {
            super(itemView);
            configurationName = itemView.findViewById(R.id.configurationName);
            configurationHost = itemView.findViewById(R.id.configurationHost);
            reorderView = itemView.findViewById(R.id.reorderView);

            item = null;

            if (!isEditMode) {
                reorderView.setVisibility(View.GONE); // disable reorder
                if (clickListener != null) {
                    itemView.setOnClickListener(v -> clickListener.onClick(item));
                }
            } else {
                if (gestureListener != null) {
                    reorderView.setOnTouchListener((v, event) -> {
                        if (event.getActionMasked() == MotionEvent.ACTION_DOWN) {
                            if (dragListener != null) {
                                dragListener.onStartDrag(this);
                            }
                        }
                        return false;
                    });
                }
            }
        }

        void bind(Configuration configuration) {
            item = configuration;
            configurationName.setText(configuration.getName());
            configurationHost.setText(configuration.getHost());
        }

        @Override
        public void onItemSelected() {
            itemView.setBackgroundColor(Color.LTGRAY);
        }

        @Override
        public void onItemClear() {
            itemView.setBackgroundColor(0);
        }
    }
}
