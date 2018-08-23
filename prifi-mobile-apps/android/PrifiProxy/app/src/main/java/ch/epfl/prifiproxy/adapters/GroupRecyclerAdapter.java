package ch.epfl.prifiproxy.adapters;

import android.content.Context;
import android.support.annotation.NonNull;
import android.support.v7.widget.RecyclerView;
import android.view.LayoutInflater;
import android.view.View;
import android.view.ViewGroup;
import android.widget.Switch;
import android.widget.TextView;

import java.util.List;

import ch.epfl.prifiproxy.R;
import ch.epfl.prifiproxy.listeners.OnCheckedListener;
import ch.epfl.prifiproxy.listeners.OnClickListener;
import ch.epfl.prifiproxy.persistence.entity.ConfigurationGroup;

public class GroupRecyclerAdapter extends RecyclerView.Adapter<GroupRecyclerAdapter.ViewHolder> {
    private List<ConfigurationGroup> dataset;
    private final OnCheckedListener checkedListener;
    private final OnClickListener clickListener;

    public GroupRecyclerAdapter(Context context, List<ConfigurationGroup> dataset,
                                OnCheckedListener checkedListener,
                                OnClickListener clickListener) {
        this.dataset = dataset;
        this.checkedListener = checkedListener;
        this.clickListener = clickListener;
    }


    @NonNull
    @Override
    public ViewHolder onCreateViewHolder(@NonNull ViewGroup parent, int viewType) {
        View v = LayoutInflater.from(parent.getContext())
                .inflate(R.layout.group_list_item, parent, false);

        return new ViewHolder(v, checkedListener, clickListener);
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
        private boolean isBound;

        ViewHolder(View itemView, OnCheckedListener checkedListener, OnClickListener clickListener) {
            super(itemView);
            groupName = itemView.findViewById(R.id.groupName);
            groupSwitch = itemView.findViewById(R.id.groupSwitch);

            if (clickListener != null) {
                itemView.setOnClickListener(v -> clickListener.onClick(getAdapterPosition()));
            }
            if (checkedListener != null) {
                groupSwitch.setOnCheckedChangeListener((buttonView, isChecked) -> {
                    if (isBound) {
                        // Stop circular calls
                        checkedListener.onChecked(getAdapterPosition(), isChecked);
                    }
                });
            }
        }

        void bind(ConfigurationGroup group) {
            isBound = false;
            groupName.setText(group.getName());
            groupSwitch.setChecked(group.isActive());
            isBound = true;
        }
    }
}
