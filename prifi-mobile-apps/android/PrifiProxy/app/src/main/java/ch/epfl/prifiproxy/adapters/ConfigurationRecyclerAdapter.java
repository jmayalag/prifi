package ch.epfl.prifiproxy.adapters;

import android.annotation.SuppressLint;
import android.support.annotation.NonNull;
import android.support.annotation.Nullable;
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
import java.util.Objects;

import ch.epfl.prifiproxy.R;
import ch.epfl.prifiproxy.listeners.OnItemClickListener;
import ch.epfl.prifiproxy.listeners.OnItemGestureListener;
import ch.epfl.prifiproxy.persistence.entity.Configuration;
import ch.epfl.prifiproxy.ui.SwipeAndDragHelper;

public class ConfigurationRecyclerAdapter extends RecyclerView.Adapter<ConfigurationRecyclerAdapter.ViewHolder>
        implements SwipeAndDragHelper.ActionCompletionContract {
    @Nullable
    private final OnItemClickListener<Configuration> clickListener;
    @Nullable
    private final OnItemGestureListener<Configuration> gestureListener;
    @NonNull
    private List<Configuration> dataset;
    @NonNull
    private final ItemTouchHelper touchHelper;

    public ConfigurationRecyclerAdapter(@Nullable OnItemClickListener<Configuration> clickListener,
                                        @Nullable OnItemGestureListener<Configuration> gestureListener) {
        this.dataset = new ArrayList<>();
        this.clickListener = clickListener;
        this.gestureListener = gestureListener;
        this.touchHelper = new ItemTouchHelper(new SwipeAndDragHelper(this));
    }

    @NonNull
    @Override
    public ViewHolder onCreateViewHolder(@NonNull ViewGroup parent, int viewType) {
        View v = LayoutInflater.from(parent.getContext())
                .inflate(R.layout.configuration_list_item, parent, false);

        return new ViewHolder(v, clickListener, gestureListener, touchHelper);
    }

    @SuppressLint("ClickableViewAccessibility")
    @Override
    public void onBindViewHolder(@NonNull ViewHolder holder, int position) {
        Configuration item = dataset.get(position);
        holder.bind(item);
        if (gestureListener != null) {
            holder.reorderView.setOnTouchListener((v, event) -> {
                if (event.getActionMasked() == MotionEvent.ACTION_DOWN) {
                    touchHelper.startDrag(holder);
                }
                return false;
            });
        }
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
    public void onViewMoved(int oldPosition, int newPosition) {
        if (gestureListener != null) {
            gestureListener.itemMoved(dataset.get(oldPosition), dataset.get(newPosition), oldPosition, newPosition);
        }
        Configuration target = dataset.get(oldPosition);
        dataset.remove(oldPosition);
        dataset.add(newPosition, target);
        notifyItemMoved(oldPosition, newPosition);
    }

    @Override
    public void onViewSwiped(int position) {
        if (gestureListener != null) {
            gestureListener.itemSwiped(dataset.get(position));
        }
    }

    static class ViewHolder extends RecyclerView.ViewHolder {
        TextView configurationName;
        TextView configurationHost;
        ImageView reorderView;
        Configuration item;

        @SuppressLint("ClickableViewAccessibility")
        ViewHolder(View itemView,
                   @Nullable OnItemClickListener<Configuration> clickListener,
                   @Nullable OnItemGestureListener<Configuration> gestureListener,
                   @NonNull ItemTouchHelper touchHelper) {
            super(itemView);
            configurationName = itemView.findViewById(R.id.configurationName);
            configurationHost = itemView.findViewById(R.id.configurationHost);
            reorderView = itemView.findViewById(R.id.reorderView);
            item = null;

            if (clickListener != null) {
                itemView.setOnClickListener(v -> clickListener.onClick(item));
            }
        }

        void bind(Configuration configuration) {
            item = configuration;
            configurationName.setText(configuration.getName());
            configurationHost.setText(configuration.getHost());
        }
    }
}
