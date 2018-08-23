package ch.epfl.prifiproxy.activities;

import android.os.Bundle;
import android.support.design.widget.FloatingActionButton;
import android.support.v7.app.AppCompatActivity;
import android.support.v7.widget.Toolbar;
import android.widget.Toast;

import ch.epfl.prifiproxy.R;

/**
 * Detail of a {@link ch.epfl.prifiproxy.persistence.entity.ConfigurationGroup}
 */
public class GroupAddActivity extends AppCompatActivity {
    protected static final String EXTRA_GROUP_ID = "groupId";

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setContentView(R.layout.activity_group_add);
        Toolbar toolbar = findViewById(R.id.toolbar);
        setSupportActionBar(toolbar);
        getSupportActionBar().setDisplayHomeAsUpEnabled(true);

        Bundle bundle = getIntent().getExtras();
        if (bundle != null) {
            int groupId = bundle.getInt(EXTRA_GROUP_ID, -1);
            Toast.makeText(this, "Got groupId: " + groupId, Toast.LENGTH_SHORT).show();
        }

        FloatingActionButton fab = findViewById(R.id.fab);
        fab.setOnClickListener(view -> addConfiguration());
    }

    private void addConfiguration() {
        Toast.makeText(this, "TODO", Toast.LENGTH_SHORT).show();
    }

}
