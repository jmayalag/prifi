package ch.epfl.prifiproxy.activities;

import android.os.Bundle;
import android.support.design.widget.FloatingActionButton;
import android.support.v7.app.AppCompatActivity;
import android.support.v7.widget.Toolbar;
import android.widget.Toast;

import ch.epfl.prifiproxy.R;
import ch.epfl.prifiproxy.persistence.entity.ConfigurationGroup;

public class GroupAddActivity extends AppCompatActivity {
    protected static final String EXTRA_GROUP_ID = "groupId";
    private ConfigurationGroup group;
    private Toolbar toolbar;

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setContentView(R.layout.activity_group_add);
        toolbar = findViewById(R.id.toolbar);
        setSupportActionBar(toolbar);
        getSupportActionBar().setDisplayHomeAsUpEnabled(true);

        Bundle bundle = getIntent().getExtras();

        if (bundle != null) {
            int groupId = bundle.getInt(EXTRA_GROUP_ID, -1);
            if (groupId != -1) {
                group = new ConfigurationGroup(groupId, "Network " + groupId, false);
                toolbar.setTitle("Edit Network");
            }
        }

        if (group == null) {
            group = new ConfigurationGroup(0, "Add Network", false);
        }

        FloatingActionButton fab = findViewById(R.id.fab);
        fab.setOnClickListener(view -> addConfiguration());
    }

    @Override
    protected void onResume() {
        super.onResume();
    }

    private void addConfiguration() {
        Toast.makeText(this, "TODO", Toast.LENGTH_SHORT).show();
    }

}
