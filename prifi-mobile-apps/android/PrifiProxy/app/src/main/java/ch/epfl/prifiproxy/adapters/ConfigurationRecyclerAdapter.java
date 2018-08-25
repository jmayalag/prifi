package ch.epfl.prifiproxy.adapters;

import android.support.annotation.NonNull;
import android.support.annotation.Nullable;
import android.support.v7.widget.RecyclerView;
import android.view.LayoutInflater;
import android.view.View;
import android.view.ViewGroup;
import android.widget.TextView;

import java.util.ArrayList;
import java.util.List;

import ch.epfl.prifiproxy.R;
import ch.epfl.prifiproxy.listeners.OnItemClickListener;
import ch.epfl.prifiproxy.persistence.entity.Configuration;

public class ConfigurationRecyclerAdapter extends RecyclerView.Adapter<ConfigurationRecyclerAdapter.ViewHolder> {
    @Nullable
    private final OnItemClickListener<Configuration> clickListener;
    @NonNull
    private List<Configuration> dataset;

    public ConfigurationRecyclerAdapter(@Nullable OnItemClickListener<Configuration> clickListener) {
        this.dataset = new ArrayList<>();
        this.clickListener = clickListener;
    }

    @NonNull
    @Override
    public ViewHolder onCreateViewHolder(@NonNull ViewGroup parent, int viewType) {
        View v = LayoutInflater.from(parent.getContext())
                .inflate(R.layout.configuration_list_item, parent, false);

        return new ViewHolder(v, clickListener);
    }

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

    static class ViewHolder extends RecyclerView.ViewHolder {
        TextView configurationName;
        Configuration item;

        ViewHolder(View itemView, @Nullable OnItemClickListener<Configuration> clickListener) {
            super(itemView);
            configurationName = itemView.findViewById(R.id.configurationName);
            item = null;

            if (clickListener != null) {
                itemView.setOnClickListener(v -> clickListener.onClick(item));
            }
        }

        void bind(Configuration configuration) {
            item = configuration;
            configurationName.setText(configuration.getName());
        }
    }
}
