package ch.epfl.prifiproxy.adapters;

import android.content.Context;
import android.net.Uri;
import android.support.annotation.NonNull;
import android.support.v7.widget.RecyclerView;
import android.util.TypedValue;
import android.view.LayoutInflater;
import android.view.View;
import android.view.ViewGroup;
import android.widget.Filter;
import android.widget.Filterable;
import android.widget.ImageView;
import android.widget.Switch;
import android.widget.TextView;

import com.bumptech.glide.Glide;
import com.bumptech.glide.load.DecodeFormat;
import com.bumptech.glide.request.RequestOptions;

import java.text.Normalizer;
import java.util.ArrayList;
import java.util.List;

import ch.epfl.prifiproxy.R;
import ch.epfl.prifiproxy.listeners.OnAppCheckedListener;
import ch.epfl.prifiproxy.persistence.entity.Configuration;
import ch.epfl.prifiproxy.persistence.entity.ConfigurationGroup;
import ch.epfl.prifiproxy.utils.AppInfo;

public class GroupRecyclerAdapter extends RecyclerView.Adapter<GroupRecyclerAdapter.ViewHolder> {
    private List<ConfigurationGroup> dataset;
    private final OnAppCheckedListener checkedListener;

    public GroupRecyclerAdapter(Context context, List<ConfigurationGroup> dataset,
                                OnAppCheckedListener checkedListener) {
        this.dataset = dataset;
        this.checkedListener = checkedListener;
    }


    @NonNull
    @Override
    public ViewHolder onCreateViewHolder(@NonNull ViewGroup parent, int viewType) {
        View v = LayoutInflater.from(parent.getContext())
                .inflate(R.layout.group_list_item, parent, false);

        return new ViewHolder(v, checkedListener);
    }

    @Override
    public void onBindViewHolder(@NonNull ViewHolder holder, int position) {
        ConfigurationGroup item = dataset.get(position);
        holder.bind(item);
    }

    @Override
    public int getItemCount() {
        return dataset.size();
    }


    static class ViewHolder extends RecyclerView.ViewHolder {
        TextView groupName;
        Switch groupSwitch;

        ViewHolder(View itemView, OnAppCheckedListener checkedListener) {
            super(itemView);
            groupName = itemView.findViewById(R.id.groupName);
            groupSwitch = itemView.findViewById(R.id.groupSwitch);

            itemView.setOnClickListener(v -> groupSwitch.toggle());
            groupSwitch.setOnCheckedChangeListener((buttonView, isChecked) ->
                    checkedListener.onChecked(getAdapterPosition(), isChecked));
        }

        void bind(ConfigurationGroup group) {
            groupName.setText(group.getName());
            groupSwitch.setChecked(group.isActive());
        }
    }
}
