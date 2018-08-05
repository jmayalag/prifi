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
import ch.epfl.prifiproxy.utils.AppInfo;

public class AppSelectionAdapter extends RecyclerView.Adapter<AppSelectionAdapter.ViewHolder> implements Filterable {
    private final OnAppCheckedListener mCheckedListener;
    private final int iconSize;
    private List<AppInfo> mDataset;
    private List<AppInfo> filteredApps;

    public AppSelectionAdapter(Context context, List<AppInfo> mDataset,
                               OnAppCheckedListener checkedListener) {
        this.mDataset = mDataset;
        this.filteredApps = mDataset;
        TypedValue typedValue = new TypedValue();

        context.getTheme().resolveAttribute(
                android.R.attr.listPreferredItemHeight, typedValue, true);
        this.mCheckedListener = checkedListener;
        int height = TypedValue.complexToDimensionPixelSize(typedValue.data,
                context.getResources().getDisplayMetrics());
        this.iconSize = Math.round(height * context.getResources().getDisplayMetrics().density + 0.5f);
    }

    public List<AppInfo> getFiltered() {
        return filteredApps;
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
        AppInfo item = filteredApps.get(position);
        holder.mAppName.setText(item.label);
        holder.mPackageName.setText(item.packageName);
        holder.mSwitchPrifi.setChecked(item.usePrifi);
        if (item.icon <= 0)
            holder.mAppIcon.setImageResource(android.R.drawable.sym_def_app_icon);
        else {
            Uri uri = Uri.parse("android.resource://" + item.packageName + "/" + item.icon);
            Glide.with(holder.itemView.getContext())
                    .applyDefaultRequestOptions(new RequestOptions().format(DecodeFormat.PREFER_RGB_565))
                    .load(uri)
                    .apply(new RequestOptions().override(iconSize, iconSize))
                    .into(holder.mAppIcon);
        }
    }

    @Override
    public int getItemCount() {
        return filteredApps.size();
    }

    @Override
    public Filter getFilter() {
        return new Filter() {
            @Override
            protected FilterResults performFiltering(CharSequence constraint) {
                String filter = Normalizer.normalize(constraint, Normalizer.Form.NFD).toLowerCase();
                if (filter.isEmpty()) {
                    filteredApps = mDataset;
                } else {
                    List<AppInfo> filtered = new ArrayList<>();
                    for (AppInfo info : mDataset) {
                        if (Normalizer.normalize(info.label, Normalizer.Form.NFD).toLowerCase().contains(filter) ||
                                Normalizer.normalize(info.packageName, Normalizer.Form.NFD).toLowerCase().contains(filter)) {
                            filtered.add(info);
                        }
                    }
                    filteredApps = filtered;
                }

                FilterResults results = new FilterResults();
                results.values = filteredApps;


                return results;
            }

            @Override
            protected void publishResults(CharSequence constraint, FilterResults results) {
                //noinspection unchecked
                filteredApps = (List<AppInfo>) results.values;
                notifyDataSetChanged();
            }
        };
    }

    static class ViewHolder extends RecyclerView.ViewHolder {
        ImageView mAppIcon;
        TextView mAppName;
        TextView mPackageName;
        Switch mSwitchPrifi;

        ViewHolder(View itemView, OnAppCheckedListener changeListener) {
            super(itemView);
            mAppIcon = itemView.findViewById(R.id.appIcon);
            mAppName = itemView.findViewById(R.id.appName);
            mPackageName = itemView.findViewById(R.id.packageName);
            mSwitchPrifi = itemView.findViewById(R.id.switchPrifi);
            mSwitchPrifi.setOnCheckedChangeListener((buttonView, isChecked) ->
                    changeListener.onChecked(getAdapterPosition(), isChecked));

            itemView.setOnClickListener(v -> mSwitchPrifi.toggle());
        }
    }
}
