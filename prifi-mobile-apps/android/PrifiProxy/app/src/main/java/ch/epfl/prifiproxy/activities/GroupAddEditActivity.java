package ch.epfl.prifiproxy.activities;

import android.content.Intent;
import android.os.Bundle;
import android.support.design.widget.FloatingActionButton;
import android.support.design.widget.TextInputEditText;
import android.support.design.widget.TextInputLayout;
import android.support.v7.app.AppCompatActivity;
import android.support.v7.widget.Toolbar;
import android.text.Editable;
import android.text.TextUtils;
import android.widget.Button;

import ch.epfl.prifiproxy.R;
import ch.epfl.prifiproxy.persistence.entity.ConfigurationGroup;

public class GroupAddEditActivity extends AppCompatActivity {
    public static final int NEW_GROUP_REQUEST_CODE = 1;
    public static final int EDIT_GROUP_REQUEST_CODE = 2;
    protected static final String EXTRA_GROUP_ID = "groupId";
    protected static final String EXTRA_GROUP_NAME = "groupName";
    private ConfigurationGroup group;
    private Toolbar toolbar;

    private TextInputLayout nameInputLayout;
    private Button saveButton;
    private TextInputEditText nameInput;

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setContentView(R.layout.activity_group_add_edit);
        toolbar = findViewById(R.id.toolbar);
        setSupportActionBar(toolbar);
        getSupportActionBar().setDisplayHomeAsUpEnabled(true);

        Bundle bundle = getIntent().getExtras();

        FloatingActionButton fab = findViewById(R.id.fab);
        fab.setOnClickListener(view -> saveGroup());

        nameInputLayout = findViewById(R.id.nameInputLayout);
        nameInput = findViewById(R.id.nameInput);
        saveButton = findViewById(R.id.saveButton);
        saveButton.setOnClickListener(v -> saveGroup());

        if (bundle != null) {
            int groupId = bundle.getInt(EXTRA_GROUP_ID, -1);
            if (groupId != -1) {
                group = new ConfigurationGroup(groupId, "Network " + groupId, false);
                toolbar.setTitle("Edit Network");
            }
        }

        bindView(group);
    }

    private void bindView(ConfigurationGroup group) {
        if (group == null) return;

        nameInput.setText(group.getName());
    }

    private void saveGroup() {
        String name = nameInput.getText().toString();
        if (TextUtils.isEmpty(name)) {
            nameInputLayout.setError("Required");
            return;
        }

        Intent intent = new Intent();
        intent.putExtra(EXTRA_GROUP_NAME, name);
        setResult(RESULT_OK, intent);
        finish();
    }

}
