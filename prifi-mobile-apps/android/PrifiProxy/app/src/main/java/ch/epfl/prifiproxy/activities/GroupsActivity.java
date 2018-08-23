package ch.epfl.prifiproxy.activities;

import android.content.Intent;
import android.os.Bundle;
import android.support.design.widget.FloatingActionButton;
import android.support.v7.app.AppCompatActivity;
import android.support.v7.widget.LinearLayoutManager;
import android.support.v7.widget.RecyclerView;
import android.support.v7.widget.Toolbar;
import android.widget.Toast;

import java.util.ArrayList;
import java.util.List;

import ch.epfl.prifiproxy.R;
import ch.epfl.prifiproxy.adapters.GroupRecyclerAdapter;
import ch.epfl.prifiproxy.listeners.OnCheckedListener;
import ch.epfl.prifiproxy.listeners.OnClickListener;
import ch.epfl.prifiproxy.persistence.entity.ConfigurationGroup;

public class GroupsActivity extends AppCompatActivity implements OnCheckedListener, OnClickListener {
    private RecyclerView recyclerView;
    private List<ConfigurationGroup> groups;
    private GroupRecyclerAdapter recyclerAdapter;
    private LinearLayoutManager layoutManager;

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setContentView(R.layout.activity_groups);
        Toolbar toolbar = findViewById(R.id.toolbar);
        setSupportActionBar(toolbar);
        getSupportActionBar().setDisplayHomeAsUpEnabled(true);

        FloatingActionButton fab = findViewById(R.id.fab);
        fab.setOnClickListener(view -> addGroup());

        groups = new ArrayList<>();

        recyclerView = findViewById(R.id.recyclerView);
        recyclerView.setHasFixedSize(true);

        layoutManager = new LinearLayoutManager(this);
        recyclerView.setLayoutManager(layoutManager);

        groups = new ArrayList<>();

        groups.add(new ConfigurationGroup(1, "Home", true));
        groups.add(new ConfigurationGroup(2, "Work", false));
        groups.add(new ConfigurationGroup(3, "Lab", false));
        groups.add(new ConfigurationGroup(4, "Public", false));

        recyclerAdapter = new GroupRecyclerAdapter(this, groups, this, this);
        recyclerView.setAdapter(recyclerAdapter);
    }

    private void addGroup() {
        Intent intent = new Intent(this, GroupAddActivity.class);
        startActivity(intent);
    }

    private void editGroup(ConfigurationGroup group) {
        int groupId = group.getId();
        Intent intent = new Intent(this, GroupAddActivity.class);
        intent.putExtra(GroupAddActivity.EXTRA_GROUP_ID, groupId);
        startActivity(intent);
    }

    private void reloadList() {
        recyclerAdapter.notifyDataSetChanged();
    }

    @Override
    public void onChecked(int position, boolean isChecked) {
        ConfigurationGroup group = groups.get(position);
        Toast.makeText(this, "Checked " + group.getName(), Toast.LENGTH_SHORT).show();

        for (ConfigurationGroup g : groups) {
            g.setActive(false);
        }
        group.setActive(isChecked);
        reloadList();
    }

    @Override
    public void onClick(int position) {
        ConfigurationGroup group = groups.get(position);
        editGroup(group);
    }
}
