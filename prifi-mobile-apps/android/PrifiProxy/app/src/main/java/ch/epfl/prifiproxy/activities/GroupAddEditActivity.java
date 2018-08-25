package ch.epfl.prifiproxy.activities;

import android.arch.lifecycle.ViewModelProviders;
import android.content.Intent;
import android.os.Bundle;
import android.support.design.widget.FloatingActionButton;
import android.support.design.widget.TextInputEditText;
import android.support.design.widget.TextInputLayout;
import android.support.v7.app.AppCompatActivity;
import android.support.v7.widget.LinearLayoutManager;
import android.support.v7.widget.RecyclerView;
import android.support.v7.widget.Toolbar;
import android.text.TextUtils;
import android.view.Menu;
import android.view.View;
import android.widget.Button;
import android.widget.Toast;

import ch.epfl.prifiproxy.R;
import ch.epfl.prifiproxy.adapters.ConfigurationRecyclerAdapter;
import ch.epfl.prifiproxy.listeners.OnItemClickListener;
import ch.epfl.prifiproxy.persistence.entity.Configuration;
import ch.epfl.prifiproxy.persistence.entity.ConfigurationGroup;
import ch.epfl.prifiproxy.viewmodel.ConfigurationViewModel;

public class GroupAddEditActivity extends AppCompatActivity implements OnItemClickListener<Configuration> {
    public static final int NEW_GROUP_REQUEST_CODE = 1;
    public static final int EDIT_GROUP_REQUEST_CODE = 2;
    protected static final String EXTRA_GROUP_ID = "groupId";
    protected static final String EXTRA_GROUP_NAME = "groupName";

    private boolean isEditMode = false;
    private int menuId;

    private ConfigurationGroup group;
    private Toolbar toolbar;

    private TextInputLayout nameInputLayout;
    private TextInputEditText nameInput;

    private ConfigurationViewModel configurationViewModel;
    private RecyclerView recyclerView;
    private LinearLayoutManager layoutManager;
    private ConfigurationRecyclerAdapter recyclerAdapter;
    private Toast currentToast;

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setContentView(R.layout.activity_group_add_edit);
        toolbar = findViewById(R.id.toolbar);
        setSupportActionBar(toolbar);
        getSupportActionBar().setDisplayHomeAsUpEnabled(true);

        FloatingActionButton fab = findViewById(R.id.fab);
        fab.setOnClickListener(view -> saveGroup());

        nameInputLayout = findViewById(R.id.nameInputLayout);
        nameInput = findViewById(R.id.nameInput);

        recyclerView = findViewById(R.id.configurationList);
        recyclerView.setHasFixedSize(true);

        layoutManager = new LinearLayoutManager(this);
        recyclerView.setLayoutManager(layoutManager);

        recyclerAdapter = new ConfigurationRecyclerAdapter(this);
        recyclerView.setAdapter(recyclerAdapter);

        configurationViewModel = ViewModelProviders.of(this).get(ConfigurationViewModel.class);

        Bundle bundle = getIntent().getExtras();

        if (bundle != null) {
            int groupId = bundle.getInt(EXTRA_GROUP_ID, -1);
            String groupName = bundle.getString(EXTRA_GROUP_NAME);
            if (groupId != -1 && groupName != null) {
                getSupportActionBar().setTitle(groupName);
                group = new ConfigurationGroup(groupId, groupName, false);
            }
        }

        if (isEditMode) {
            menuId = R.menu.menu_group_add_edit;
            nameInputLayout.setVisibility(View.VISIBLE);
        } else {
            menuId = R.menu.menu_group_detail;
            nameInputLayout.setVisibility(View.GONE);
        }

        bindView(group);
    }

    @Override
    public boolean onCreateOptionsMenu(Menu menu) {
        getMenuInflater().inflate(menuId, menu);

        return true;
    }

    private void bindView(ConfigurationGroup group) {
        if (group == null) return;

        nameInput.setText(group.getName());
        configurationViewModel.setGroupId(group.getId());
        configurationViewModel.getConfigurations().observe(this, recyclerAdapter::setData);
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

    @Override
    public void onClick(Configuration item) {
        if (currentToast != null) {
            currentToast.cancel();
            currentToast = null;
        }
        currentToast = Toast.makeText(this, "Clicked " + item.getName(), Toast.LENGTH_SHORT);
        currentToast.show();
    }
}
