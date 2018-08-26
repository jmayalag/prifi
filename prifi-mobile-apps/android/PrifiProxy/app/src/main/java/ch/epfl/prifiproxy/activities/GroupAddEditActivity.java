package ch.epfl.prifiproxy.activities;

import android.arch.lifecycle.ViewModelProviders;
import android.content.Context;
import android.content.Intent;
import android.os.Bundle;
import android.support.design.widget.FloatingActionButton;
import android.support.design.widget.TextInputEditText;
import android.support.design.widget.TextInputLayout;
import android.support.v7.app.AlertDialog;
import android.support.v7.app.AppCompatActivity;
import android.support.v7.widget.LinearLayoutManager;
import android.support.v7.widget.RecyclerView;
import android.support.v7.widget.Toolbar;
import android.text.TextUtils;
import android.util.Log;
import android.view.Menu;
import android.view.MenuItem;
import android.view.View;
import android.widget.Toast;

import java.util.Objects;

import ch.epfl.prifiproxy.R;
import ch.epfl.prifiproxy.adapters.ConfigurationRecyclerAdapter;
import ch.epfl.prifiproxy.listeners.OnItemClickListener;
import ch.epfl.prifiproxy.listeners.OnItemGestureListener;
import ch.epfl.prifiproxy.persistence.entity.Configuration;
import ch.epfl.prifiproxy.persistence.entity.ConfigurationGroup;
import ch.epfl.prifiproxy.ui.Mode;
import ch.epfl.prifiproxy.viewmodel.ConfigurationViewModel;

public class GroupAddEditActivity extends AppCompatActivity implements OnItemClickListener<Configuration>, OnItemGestureListener<Configuration> {
    private static final String TAG = "GROUP_ADD_EDIT_ACTIVITY";
    protected static final String EXTRA_GROUP_ID = "groupId";
    private static final String EXTRA_MODE = "mode";

    private int mode;
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
    private AlertDialog.Builder deleteDialogBuilder;

    private static Intent getIntent(Context packageContext) {
        return new Intent(packageContext, GroupAddEditActivity.class);
    }

    public static Intent intentAdd(Context packageContext) {
        Intent intent = getIntent(packageContext);
        intent.putExtra(EXTRA_MODE, Mode.ADD);
        return intent;
    }

    public static Intent intentEdit(Context packageContext, ConfigurationGroup group) {
        Intent intent = getIntent(packageContext);
        intent.putExtra(EXTRA_MODE, Mode.EDIT);
        intent.putExtra(EXTRA_GROUP_ID, group.getId());
        return intent;
    }

    public static Intent intentDetail(Context packageContext, ConfigurationGroup group) {
        Intent intent = getIntent(packageContext);
        intent.putExtra(EXTRA_MODE, Mode.DETAIL);
        intent.putExtra(EXTRA_GROUP_ID, group.getId());
        return intent;
    }

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setContentView(R.layout.activity_group_add_edit);
        toolbar = findViewById(R.id.toolbar);
        setSupportActionBar(toolbar);
        getSupportActionBar().setDisplayHomeAsUpEnabled(true);

        FloatingActionButton fab = findViewById(R.id.fab);
        fab.setOnClickListener(view -> addConfiguration());

        nameInputLayout = findViewById(R.id.nameInputLayout);
        nameInput = findViewById(R.id.nameInput);

        recyclerView = findViewById(R.id.configurationList);
        recyclerView.setHasFixedSize(true);

        layoutManager = new LinearLayoutManager(this);
        recyclerView.setLayoutManager(layoutManager);

        recyclerAdapter = new ConfigurationRecyclerAdapter(this, this);
        recyclerView.setAdapter(recyclerAdapter);

        configurationViewModel = ViewModelProviders.of(this).get(ConfigurationViewModel.class);

        Bundle bundle = getIntent().getExtras();

        if (bundle == null) {
            // Default is adding mode
            mode = Mode.ADD;
        } else {
            mode = bundle.getInt(EXTRA_MODE, Mode.ADD);
            int groupId = bundle.getInt(EXTRA_GROUP_ID, -1);
            if (groupId != -1) {
                configurationViewModel.init(groupId);
                configurationViewModel.getGroup().observe(this, this::bindView);
            }
        }

        if (mode == Mode.ADD || mode == Mode.EDIT) {
            menuId = R.menu.menu_group_add_edit;
            nameInputLayout.setVisibility(View.VISIBLE);
            fab.setVisibility(View.GONE);
        } else {
            menuId = R.menu.menu_group_detail;
            nameInputLayout.setVisibility(View.GONE);
        }

        bindView(group);

        deleteDialogBuilder = new AlertDialog.Builder(this)
                .setTitle(R.string.dialog_delete_group_title)
                .setMessage(R.string.dialog_delete_group_msg)
                .setCancelable(true)
                .setNegativeButton(android.R.string.no, null)
                .setPositiveButton(android.R.string.yes, (dialog, which) -> deleteGroup());
    }

    @Override
    public boolean onCreateOptionsMenu(Menu menu) {
        getMenuInflater().inflate(menuId, menu);

        return true;
    }

    @Override
    public boolean onOptionsItemSelected(MenuItem item) {
        int id = item.getItemId();

        switch (id) {
            case android.R.id.home:
                finish(); //TODO: Fix up navigation
                return true;
            case R.id.save_group:
                saveGroup();
                return true;
            case R.id.edit_group:
                editGroup();
                return true;
            case R.id.delete_group:
                deleteDialogBuilder.show();
                return true;
        }

        return super.onOptionsItemSelected(item);
    }

    private void bindView(ConfigurationGroup group) {
        if (group == null) {
            return;
        }

        this.group = group;
        configurationViewModel.getConfigurations().observe(this, recyclerAdapter::setData);
        nameInput.setText(group.getName());
        Objects.requireNonNull(getSupportActionBar()).setTitle(group.getName());
    }

    private void saveGroup() {
        ConfigurationGroup group = this.group;
        if (group == null) {
            // Add a new group
            group = new ConfigurationGroup();
        }

        String name = nameInput.getText().toString();
        if (TextUtils.isEmpty(name)) {
            nameInputLayout.setError(getString(R.string.required));
            return;
        }
        group.setName(name);

        configurationViewModel.insertOrUpdate(group);
        finish();
    }

    private void editGroup() {
        if (group == null) {
            Log.e(TAG, "Illegal state. Can't edit null group");
            finish();
        }
        startActivity(intentEdit(this, group));
    }

    private void deleteGroup() {
        configurationViewModel.delete(group);
        finish();
    }

    private void showConfiguration(Configuration configuration) {
        Intent intent = ConfigurationAddEditActivity.intentDetail(this, configuration);
        startActivity(intent);
    }

    private void addConfiguration() {
        Intent intent = ConfigurationAddEditActivity.intentAdd(this, group);
        startActivity(intent);
    }

    @Override
    public void onClick(Configuration item) {
        showConfiguration(item);
    }

    @Override
    public void itemMoved(Configuration item, Configuration afterItem, int fromPosition, int toPosition) {
        Log.i(TAG, "Moved " + item.getName() + " from " + fromPosition + " to " + toPosition);
    }

    @Override
    public void itemSwiped(Configuration item) {
        Log.i(TAG, "Swiped " + item.getName());
    }
}
