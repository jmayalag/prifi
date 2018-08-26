package ch.epfl.prifiproxy.activities;

import android.arch.lifecycle.ViewModelProviders;
import android.content.Intent;
import android.os.Bundle;
import android.support.design.widget.FloatingActionButton;
import android.support.v7.app.AppCompatActivity;
import android.support.v7.widget.LinearLayoutManager;
import android.support.v7.widget.RecyclerView;
import android.support.v7.widget.Toolbar;

import ch.epfl.prifiproxy.R;
import ch.epfl.prifiproxy.adapters.GroupRecyclerAdapter;
import ch.epfl.prifiproxy.listeners.OnItemCheckedListener;
import ch.epfl.prifiproxy.listeners.OnItemClickListener;
import ch.epfl.prifiproxy.persistence.entity.ConfigurationGroup;
import ch.epfl.prifiproxy.viewmodel.ConfigurationGroupViewModel;

public class GroupsActivity extends AppCompatActivity
        implements OnItemCheckedListener<ConfigurationGroup>, OnItemClickListener<ConfigurationGroup> {
    private RecyclerView recyclerView;
    private GroupRecyclerAdapter recyclerAdapter;
    private LinearLayoutManager layoutManager;
    private ConfigurationGroupViewModel groupViewModel;

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setContentView(R.layout.activity_groups);
        Toolbar toolbar = findViewById(R.id.toolbar);
        setSupportActionBar(toolbar);
        getSupportActionBar().setDisplayHomeAsUpEnabled(true);

        FloatingActionButton fab = findViewById(R.id.fab);
        fab.setOnClickListener(view -> addGroup());


        recyclerView = findViewById(R.id.recyclerView);
        recyclerView.setHasFixedSize(true);

        layoutManager = new LinearLayoutManager(this);
        recyclerView.setLayoutManager(layoutManager);

        recyclerAdapter = new GroupRecyclerAdapter(this, this);
        recyclerView.setAdapter(recyclerAdapter);

        groupViewModel = ViewModelProviders.of(this).get(ConfigurationGroupViewModel.class);
        groupViewModel.getAllGroups().observe(this, recyclerAdapter::setData);
    }

    private void addGroup() {
        startActivity(GroupAddEditActivity.intentAdd(this));
    }

    private void detailGroup(ConfigurationGroup group) {
        startActivity(GroupAddEditActivity.intentDetail(this, group));
    }

    @Override
    public void onChecked(ConfigurationGroup item, boolean isChecked) {
        groupViewModel.setActiveGroup(item, isChecked);
    }

    @Override
    public void onClick(ConfigurationGroup item) {
        detailGroup(item);
    }
}
