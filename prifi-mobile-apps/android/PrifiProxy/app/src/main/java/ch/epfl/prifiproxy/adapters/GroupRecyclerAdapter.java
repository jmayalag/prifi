package ch.epfl.prifiproxy.adapters;

import android.support.annotation.NonNull;
import android.support.annotation.Nullable;
import android.support.v7.widget.RecyclerView;
import android.view.LayoutInflater;
import android.view.View;
import android.view.ViewGroup;
import android.widget.Switch;
import android.widget.TextView;

import java.util.ArrayList;
import java.util.List;

import ch.epfl.prifiproxy.R;
import ch.epfl.prifiproxy.listeners.OnItemCheckedListener;
import ch.epfl.prifiproxy.listeners.OnItemClickListener;
import ch.epfl.prifiproxy.persistence.entity.ConfigurationGroup;

public class GroupRecyclerAdapter extends RecyclerView.Adapter<GroupRecyclerAdapter.ViewHolder> {

    @Nullable
    private final OnItemCheckedListener<ConfigurationGroup> checkedListener;
    @Nullable
    private final OnItemClickListener<ConfigurationGroup> clickListener;
    @NonNull
    private List<ConfigurationGroup> dataset;

    public GroupRecyclerAdapter(@Nullable OnItemCheckedListener<ConfigurationGroup> checkedListener,
                                @Nullable OnItemClickListener<ConfigurationGroup> clickListener) {
        this.dataset = new ArrayList<>();
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

    public void setData(@NonNull List<ConfigurationGroup> dataset) {
        this.dataset = dataset;
        notifyDataSetChanged();
    }

    static class ViewHolder extends RecyclerView.ViewHolder {
        TextView groupName;
        Switch groupSwitch;
        private boolean isBound;
        ConfigurationGroup item;

        ViewHolder(View itemView,
                   @Nullable OnItemCheckedListener<ConfigurationGroup> checkedListener,
                   @Nullable OnItemClickListener<ConfigurationGroup> clickListener) {
            super(itemView);
            groupName = itemView.findViewById(R.id.groupName);
            groupSwitch = itemView.findViewById(R.id.groupSwitch);
            item = null;

            if (clickListener != null) {
                itemView.setOnClickListener(v -> clickListener.onClick(item));
            }
            if (checkedListener != null) {
                groupSwitch.setOnCheckedChangeListener((buttonView, isChecked) -> {
                    if (isBound) {
                        // Stop circular calls
                        checkedListener.onChecked(item, isChecked);
                    }
                });
            }
        }

        void bind(ConfigurationGroup group) {
            isBound = false;
            item = group;
            groupName.setText(group.getName());
            groupSwitch.setChecked(group.isActive());
            isBound = true;
        }
    }
}
