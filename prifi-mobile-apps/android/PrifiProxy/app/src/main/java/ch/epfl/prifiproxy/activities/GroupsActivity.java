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

    @Override
    protected void onActivityResult(int requestCode, int resultCode, Intent data) {
        super.onActivityResult(requestCode, resultCode, data);

        if (requestCode == GroupAddEditActivity.NEW_GROUP_REQUEST_CODE && resultCode == RESULT_OK) {
            String name = data.getStringExtra(GroupAddEditActivity.EXTRA_GROUP_NAME);
            ConfigurationGroup group = new ConfigurationGroup(0, name, false);
            groupViewModel.insert(group);
        }
    }

    private void addGroup() {
        Intent intent = new Intent(this, GroupAddEditActivity.class);
        startActivityForResult(intent, GroupAddEditActivity.NEW_GROUP_REQUEST_CODE);
    }

    private void editGroup(ConfigurationGroup group) {
        Intent intent = new Intent(this, GroupAddEditActivity.class);
        intent.putExtra(GroupAddEditActivity.EXTRA_GROUP_ID, group.getId());
        intent.putExtra(GroupAddEditActivity.EXTRA_GROUP_NAME, group.getName());
        startActivity(intent);
    }

    @Override
    public void onChecked(ConfigurationGroup item, boolean isChecked) {

    }

    @Override
    public void onClick(ConfigurationGroup item) {
        editGroup(item);
    }
}
