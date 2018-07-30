package ch.epfl.prifiproxy.adapters;

import android.support.annotation.NonNull;
import android.support.v7.widget.RecyclerView;
import android.view.LayoutInflater;
import android.view.View;
import android.view.ViewGroup;
import android.widget.Switch;
import android.widget.TextView;

import java.util.List;

import ch.epfl.prifiproxy.R;
import ch.epfl.prifiproxy.listeners.OnAppCheckedListener;
import ch.epfl.prifiproxy.utils.AppInfo;

public class AppSelectionAdapter extends RecyclerView.Adapter<AppSelectionAdapter.ViewHolder> {
    private final OnAppCheckedListener mCheckedListener;
    private List<AppInfo> mDataset;

    public AppSelectionAdapter(List<AppInfo> mDataset,
                               OnAppCheckedListener checkedListener) {
        this.mDataset = mDataset;
        this.mCheckedListener = checkedListener;
    }

    @NonNull
    @Override
    public ViewHolder onCreateViewHolder(@NonNull ViewGroup parent, int viewType) {
        View v = LayoutInflater.from(parent.getContext())
                .inflate(R.layout.app_item, parent, false);

        return new ViewHolder(v, mCheckedListener);
    }

    @Override
    public void onBindViewHolder(@NonNull ViewHolder holder, int position) {
        AppInfo item = mDataset.get(position);
        holder.mAppName.setText(item.label);
        holder.mPackageName.setText(item.packageName);
        holder.mSwitchPrifi.setChecked(item.usePrifi);
    }

    @Override
    public int getItemCount() {
        return mDataset.size();
    }

    static class ViewHolder extends RecyclerView.ViewHolder {
        TextView mAppName;
        TextView mPackageName;
        Switch mSwitchPrifi;

        ViewHolder(View itemView, OnAppCheckedListener changeListener) {
            super(itemView);
            mAppName = itemView.findViewById(R.id.appName);
            mPackageName = itemView.findViewById(R.id.packageName);
            mSwitchPrifi = itemView.findViewById(R.id.switchPrifi);
            mSwitchPrifi.setOnCheckedChangeListener((buttonView, isChecked) ->
                    changeListener.onChecked(getAdapterPosition(), isChecked));
        }
    }
}
